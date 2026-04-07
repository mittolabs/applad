package workflows

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/dop251/goja"
	_ "github.com/go-sql-driver/mysql"
)

// SubWorkflowRunner is set by the service to allow sub-workflow execution.
// It loads a workflow by ID and executes it with the given trigger data.
var SubWorkflowRunner func(ctx context.Context, workflowID string, triggerData map[string]interface{}, depth int) ([]StepLog, interface{}, error)

const maxSubWorkflowDepth = 5

// RunWorkflow executes a workflow and returns step logs.
func RunWorkflow(ctx context.Context, wf *Workflow, triggerData map[string]interface{}) ([]StepLog, error) {
	if len(wf.Nodes) == 0 {
		return []StepLog{}, nil
	}

	// Read retry settings from the workflow
	retryAttempts := wf.RetryAttempts
	retryDelayMs := wf.RetryDelayMs
	if retryDelayMs <= 0 {
		retryDelayMs = 1000 // default 1s between retries
	}

	// Build node lookup and adjacency list
	nodeMap := make(map[string]*Node, len(wf.Nodes))
	for i := range wf.Nodes {
		nodeMap[wf.Nodes[i].ID] = &wf.Nodes[i]
	}

	order := topoSort(wf.Nodes, wf.Edges)

	// Build try_catch scope: map tryNode IDs to their catch target
	tryCatchScope := map[string]string{}  // tryNodeID -> catchTarget
	tryCatchNodes := map[string][]string{} // tryCatchNodeID -> tryNodeIDs
	for i := range wf.Nodes {
		if wf.Nodes[i].Type == "try_catch" {
			catchTarget, _ := wf.Nodes[i].Config["catchTarget"].(string)
			tryNodeIDs := extractStringSlice(wf.Nodes[i].Config, "tryNodes")
			tryCatchNodes[wf.Nodes[i].ID] = tryNodeIDs
			for _, tnID := range tryNodeIDs {
				tryCatchScope[tnID] = catchTarget
			}
		}
	}

	// Execution context: data passed between nodes
	execCtx := map[string]interface{}{
		"trigger": triggerData,
	}

	var logs []StepLog
	skipped := map[string]bool{}

	for _, nodeID := range order {
		node := nodeMap[nodeID]
		if node == nil {
			continue
		}

		// Check if this node was skipped by an upstream condition
		if skipped[nodeID] {
			logs = append(logs, StepLog{
				NodeID: node.ID, NodeType: node.Type, Label: node.Label,
				Status: "skipped",
			})
			continue
		}

		start := time.Now()
		input := cloneMap(execCtx)

		var output interface{}
		var skipTargets []string
		var err error

		// Execute with retry logic
		maxAttempts := 1
		if retryAttempts > 0 {
			maxAttempts = retryAttempts + 1 // retryAttempts is the number of *retries*, not total attempts
		}

		for attempt := 0; attempt < maxAttempts; attempt++ {
			output, skipTargets, err = executeNode(ctx, node, execCtx)
			if err == nil {
				break
			}
			// If there are remaining retries, wait before the next attempt
			if attempt < maxAttempts-1 {
				delay := time.Duration(retryDelayMs) * time.Millisecond
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					err = ctx.Err()
					break
				}
			}
		}

		duration := time.Since(start).Milliseconds()

		stepLog := StepLog{
			NodeID:     node.ID,
			NodeType:   node.Type,
			Label:      node.Label,
			Input:      input,
			Output:     output,
			DurationMs: duration,
		}

		if err != nil {
			stepLog.Status = "failed"
			stepLog.Error = err.Error()
			logs = append(logs, stepLog)

			// Check if this node is inside a try_catch scope
			if catchTarget, inTryCatch := tryCatchScope[nodeID]; inTryCatch && catchTarget != "" {
				// Store the error in execution context for the catch handler
				execCtx[nodeID] = map[string]interface{}{"error": err.Error()}
				execCtx["_lastError"] = err.Error()

				// Skip remaining tryNodes that belong to the same try_catch group
				for tcNodeID, tryIDs := range tryCatchNodes {
					for _, tid := range tryIDs {
						if tid == nodeID {
							// Found the try_catch group — skip remaining tryNodes
							for _, skipID := range tryCatchNodes[tcNodeID] {
								if skipID != nodeID {
									skipped[skipID] = true
								}
							}
							break
						}
					}
				}

				// Un-skip the catch target (the try_catch node would have skipped it on success)
				delete(skipped, catchTarget)
				continue
			}

			return logs, fmt.Errorf("node %s (%s) failed: %w", node.ID, node.Label, err)
		}

		stepLog.Status = "completed"
		logs = append(logs, stepLog)

		// Store node output in context
		execCtx[node.ID] = output

		// Mark nodes to skip (from if_condition false branch)
		for _, t := range skipTargets {
			skipped[t] = true
		}
	}

	return logs, nil
}

// executeNode runs a single node and returns its output.
// skipTargets is non-nil only for if_condition nodes — contains node IDs to skip.
func executeNode(ctx context.Context, node *Node, execCtx map[string]interface{}) (output interface{}, skipTargets []string, err error) {
	switch node.Type {
	case "http_request":
		output, err = execHTTPRequest(ctx, node.Config, execCtx)
	case "send_email":
		output, err = execSendEmail(node.Config, execCtx)
	case "set_variable":
		output, err = execSetVariable(node.Config, execCtx)
	case "code":
		output, err = execCode(node.Config, execCtx)
	case "javascript":
		output, err = execJavaScript(node.Config, execCtx)
	case "if_condition":
		output, skipTargets, err = execIfCondition(node.Config, execCtx)
	case "delay":
		output, err = execDelay(ctx, node.Config)
	// ── Flow nodes ──
	case "switch":
		output, skipTargets, err = execSwitch(node.Config, execCtx)
	case "merge":
		output = map[string]interface{}{"merged": true}
	case "loop":
		output, err = execLoop(node.Config, execCtx)
	case "wait":
		output, err = execWait(ctx, node.Config)
	case "no_operation":
		output = map[string]interface{}{}
	case "execute_sub_workflow":
		wfID, _ := node.Config["workflowId"].(string)
		if wfID == "" {
			output = map[string]interface{}{"error": "workflowId is required"}
		} else if SubWorkflowRunner == nil {
			output = map[string]interface{}{"subWorkflowId": wfID, "status": "runner_not_configured"}
		} else {
			depth, _ := execCtx["__depth__"].(int)
			if depth >= maxSubWorkflowDepth {
				err = fmt.Errorf("execute_sub_workflow: max recursion depth %d exceeded", maxSubWorkflowDepth)
			} else {
				subTrigger := map[string]interface{}{"parent": execCtx}
				_, subOutput, subErr := SubWorkflowRunner(ctx, wfID, subTrigger, depth+1)
				if subErr != nil {
					err = fmt.Errorf("execute_sub_workflow: %w", subErr)
				} else {
					output = map[string]interface{}{"subWorkflowId": wfID, "result": subOutput}
				}
			}
		}
	case "filter":
		output, skipTargets, err = execFilter(node.Config, execCtx)
	case "try_catch":
		output, skipTargets, err = execTryCatch(node.Config, execCtx)
	case "stop_and_error":
		output, err = execStopAndError(node.Config, execCtx)
	// ── Data transform nodes ──
	case "edit_fields":
		output, err = execEditFields(node.Config, execCtx)
	case "aggregate":
		output, err = execAggregate(node.Config, execCtx)
	case "summarize":
		output, err = execSummarize(node.Config, execCtx)
	case "limit":
		cnt, _ := node.Config["count"].(float64)
		output = map[string]interface{}{"limit": int(cnt)}
	case "split_out":
		output, err = execSplitOut(node.Config, execCtx)
	case "remove_duplicates":
		output, err = execRemoveDuplicates(node.Config, execCtx)
	case "date_time":
		output, err = execDateTime(node.Config, execCtx)
	case "html_parse":
		html, _ := node.Config["html"].(string)
		output = map[string]interface{}{"result": resolveTemplate(html, execCtx)}
	case "crypto":
		output, err = execCrypto(node.Config, execCtx)
	case "convert_to_json":
		data := resolveFieldValue(node.Config["data"].(string), execCtx)
		b, _ := json.Marshal(data)
		output = map[string]interface{}{"json": string(b)}
	case "extract_from_json":
		output, err = execExtractJSON(node.Config, execCtx)
	// ── Integration nodes ──
	case "slack":
		output, err = execWebhookPost(ctx, node.Config, execCtx, "slack")
	case "discord":
		output, err = execWebhookPost(ctx, node.Config, execCtx, "discord")
	case "telegram":
		output, err = execTelegram(ctx, node.Config, execCtx)
	case "github":
		output, err = execGitHub(ctx, node.Config, execCtx)
	case "google_sheets":
		output, err = execGoogleSheets(ctx, node.Config, execCtx)
	case "notion":
		output, err = execNotion(ctx, node.Config, execCtx)
	case "stripe":
		output, err = execStripe(ctx, node.Config, execCtx)
	case "twilio_sms":
		output, err = execTwilioSMS(ctx, node.Config, execCtx)
	case "postgres_query":
		output, err = execPostgresQuery(ctx, node.Config, execCtx)
	case "mysql_query":
		output, err = execMySQLQuery(ctx, node.Config, execCtx)
	case "redis_command":
		output, err = execRedisCommand(ctx, node.Config, execCtx)
	case "s3":
		output, err = execS3(ctx, node.Config, execCtx)
	case "sendgrid":
		output, err = execSendGrid(ctx, node.Config, execCtx)
	case "jira":
		output, err = execJira(ctx, node.Config, execCtx)
	// ── AI nodes ──
	case "ai_transform":
		output, err = execAITransform(ctx, node.Config, execCtx)
	case "ai_summarize":
		output, err = execAISummarize(ctx, node.Config, execCtx)
	case "ai_agent":
		output, err = execAIAgent(ctx, node.Config, execCtx)
	// ── Applad-native nodes ──
	case "applad_auth":
		output, err = execApplad(ctx, node.Config, execCtx, "auth")
	case "applad_database":
		output, err = execApplad(ctx, node.Config, execCtx, "database")
	case "applad_storage":
		output, err = execApplad(ctx, node.Config, execCtx, "storage")
	case "applad_functions":
		output, err = execApplad(ctx, node.Config, execCtx, "functions")
	case "applad_messaging":
		output, err = execApplad(ctx, node.Config, execCtx, "messaging")
	// ── Additional flow nodes ──
	case "sort":
		output, err = execSort(node.Config, execCtx)
	case "rename_keys":
		output, err = execRenameKeys(node.Config, execCtx)
	case "compare_datasets":
		output, err = execCompareDatasets(node.Config, execCtx)
	default:
		output = map[string]interface{}{"message": fmt.Sprintf("unknown node type: %s", node.Type)}
	}
	return
}

// --- Node Implementations ---

func execHTTPRequest(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	method, _ := config["method"].(string)
	if method == "" {
		method = "GET"
	}
	url, _ := config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("http_request: url is required")
	}

	url = resolveTemplate(url, execCtx)

	var body io.Reader
	if bodyStr, ok := config["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(resolveTemplate(bodyStr, execCtx))
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, body)
	if err != nil {
		return nil, fmt.Errorf("http_request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http_request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit

	result := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    flattenHeaders(resp.Header),
	}

	var jsonBody interface{}
	if json.Unmarshal(respBody, &jsonBody) == nil {
		result["body"] = jsonBody
	} else {
		result["body"] = string(respBody)
	}

	return result, nil
}

func execSendEmail(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	host, _ := config["smtpHost"].(string)
	port, _ := config["smtpPort"].(string)
	user, _ := config["smtpUser"].(string)
	pass, _ := config["smtpPass"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)

	if host == "" || to == "" || subject == "" {
		return nil, fmt.Errorf("send_email: smtpHost, to, and subject are required")
	}
	if port == "" {
		port = "587"
	}

	subject = resolveTemplate(subject, execCtx)
	body = resolveTemplate(body, execCtx)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	err := smtp.SendMail(host+":"+port, auth, from, strings.Split(to, ","), []byte(msg))
	if err != nil {
		return nil, fmt.Errorf("send_email: %w", err)
	}

	return map[string]interface{}{"status": "sent", "to": to}, nil
}

func execSetVariable(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	key, _ := config["key"].(string)
	value := config["value"]

	if key == "" {
		return nil, fmt.Errorf("set_variable: key is required")
	}

	if strVal, ok := value.(string); ok {
		value = resolveTemplate(strVal, execCtx)
	}

	execCtx[key] = value
	return map[string]interface{}{key: value}, nil
}

func execCode(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	expression, _ := config["expression"].(string)
	if expression == "" {
		return nil, fmt.Errorf("code: expression is required")
	}

	result := resolveTemplate(expression, execCtx)
	return map[string]interface{}{"result": result}, nil
}

func execIfCondition(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, []string, error) {
	field, _ := config["field"].(string)
	operator, _ := config["operator"].(string)
	value := config["value"]
	trueBranch, _ := config["trueBranch"].(string)  // node ID for true
	falseBranch, _ := config["falseBranch"].(string) // node ID for false

	// Resolve the field value from execution context
	fieldValue := resolveFieldValue(field, execCtx)

	matched := evaluateCondition(fieldValue, operator, value)

	result := map[string]interface{}{
		"field":    field,
		"operator": operator,
		"value":    value,
		"actual":   fieldValue,
		"matched":  matched,
	}

	var skip []string
	if matched {
		if falseBranch != "" {
			skip = append(skip, falseBranch)
		}
	} else {
		if trueBranch != "" {
			skip = append(skip, trueBranch)
		}
	}

	return result, skip, nil
}

func execDelay(ctx context.Context, config map[string]interface{}) (interface{}, error) {
	ms, _ := config["durationMs"].(float64)
	if ms <= 0 {
		ms = 1000
	}
	if ms > 300000 { // cap at 5 minutes
		ms = 300000
	}

	d := time.Duration(ms) * time.Millisecond
	select {
	case <-time.After(d):
		return map[string]interface{}{"delayed": d.String()}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// --- Flow Node Implementations ---

func execSwitch(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, []string, error) {
	field, _ := config["field"].(string)
	defaultTarget, _ := config["defaultTarget"].(string)

	fieldValue := fmt.Sprintf("%v", resolveFieldValue(field, execCtx))

	cases, _ := config["cases"].([]interface{})
	var matched string
	var allTargets []string

	for _, c := range cases {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		val := fmt.Sprintf("%v", cm["value"])
		target, _ := cm["targetNodeId"].(string)
		allTargets = append(allTargets, target)
		if val == fieldValue {
			matched = target
		}
	}

	// Also check simple case1/case2 config (from UI)
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("case%d", i)
		val, ok := config[key].(string)
		if !ok || val == "" {
			break
		}
		target, _ := config[fmt.Sprintf("case%dTarget", i)].(string)
		if target != "" {
			allTargets = append(allTargets, target)
		}
		if val == fieldValue && matched == "" {
			matched = target
		}
	}

	// Skip all non-matching branches
	var skip []string
	if matched != "" {
		for _, t := range allTargets {
			if t != matched && t != "" {
				skip = append(skip, t)
			}
		}
	} else if defaultTarget != "" {
		for _, t := range allTargets {
			if t != defaultTarget && t != "" {
				skip = append(skip, t)
			}
		}
	}

	return map[string]interface{}{
		"field":   field,
		"value":   fieldValue,
		"matched": matched != "",
	}, skip, nil
}

func execLoop(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	itemsField, _ := config["items"].(string)
	loopVar, _ := config["loopVariable"].(string)
	if loopVar == "" {
		loopVar = "item"
	}

	items := resolveFieldValue(itemsField, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"items": []interface{}{}, "count": 0}, nil
	}

	execCtx[loopVar+"s"] = arr
	if len(arr) > 0 {
		execCtx[loopVar] = arr[0]
	}

	return map[string]interface{}{"items": arr, "count": len(arr)}, nil
}

func execWait(ctx context.Context, config map[string]interface{}) (interface{}, error) {
	secs, _ := config["seconds"].(float64)
	if secs <= 0 {
		secs = 1
	}
	if secs > 300 {
		secs = 300
	}
	d := time.Duration(secs * float64(time.Second))
	select {
	case <-time.After(d):
		return map[string]interface{}{"waited": d.String()}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func execFilter(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, []string, error) {
	field, _ := config["field"].(string)
	operator, _ := config["operator"].(string)
	value := config["value"]

	fieldValue := resolveFieldValue(field, execCtx)
	matched := evaluateCondition(fieldValue, operator, value)

	result := map[string]interface{}{
		"field":    field,
		"operator": operator,
		"matched":  matched,
	}

	// If not matched, skip all downstream (empty skip = pass through)
	// The caller handles skipping via the edge targets
	if !matched {
		return result, []string{"__all_downstream__"}, nil
	}
	return result, nil, nil
}

// --- Data Transform Implementations ---

func execEditFields(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		// Try parsing from JSON string
		fieldsStr, _ := config["fields"].(string)
		if fieldsStr != "" {
			fieldsStr = resolveTemplate(fieldsStr, execCtx)
			if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
				return nil, fmt.Errorf("edit_fields: invalid JSON: %w", err)
			}
		} else {
			fields = map[string]interface{}{}
		}
	}
	for k, v := range fields {
		if strVal, ok := v.(string); ok {
			v = resolveTemplate(strVal, execCtx)
		}
		execCtx[k] = v
	}
	return fields, nil
}

func execAggregate(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field, _ := config["field"].(string)
	operation, _ := config["operation"].(string)

	items := resolveFieldValue(field, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"result": 0}, nil
	}

	switch operation {
	case "count":
		return map[string]interface{}{"result": len(arr)}, nil
	case "sum", "avg", "min", "max":
		var nums []float64
		for _, v := range arr {
			switch n := v.(type) {
			case float64:
				nums = append(nums, n)
			case int:
				nums = append(nums, float64(n))
			case string:
				if f, err := strconv.ParseFloat(n, 64); err == nil {
					nums = append(nums, f)
				}
			}
		}
		if len(nums) == 0 {
			return map[string]interface{}{"result": 0}, nil
		}
		switch operation {
		case "sum":
			s := 0.0
			for _, n := range nums {
				s += n
			}
			return map[string]interface{}{"result": s}, nil
		case "avg":
			s := 0.0
			for _, n := range nums {
				s += n
			}
			return map[string]interface{}{"result": s / float64(len(nums))}, nil
		case "min":
			m := nums[0]
			for _, n := range nums[1:] {
				m = math.Min(m, n)
			}
			return map[string]interface{}{"result": m}, nil
		case "max":
			m := nums[0]
			for _, n := range nums[1:] {
				m = math.Max(m, n)
			}
			return map[string]interface{}{"result": m}, nil
		}
	}
	return map[string]interface{}{"result": len(arr)}, nil
}

func execSummarize(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field, _ := config["field"].(string)
	groupBy, _ := config["groupBy"].(string)

	items := resolveFieldValue(field, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"groups": map[string]int{}}, nil
	}

	groups := map[string]int{}
	for _, item := range arr {
		key := "unknown"
		if m, ok := item.(map[string]interface{}); ok && groupBy != "" {
			key = fmt.Sprintf("%v", m[groupBy])
		} else {
			key = fmt.Sprintf("%v", item)
		}
		groups[key]++
	}

	return map[string]interface{}{"groups": groups, "totalGroups": len(groups)}, nil
}

func execSplitOut(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field, _ := config["field"].(string)
	items := resolveFieldValue(field, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"items": []interface{}{}}, nil
	}
	return map[string]interface{}{"items": arr, "count": len(arr)}, nil
}

func execRemoveDuplicates(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field, _ := config["field"].(string)
	items := resolveFieldValue(field, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"items": []interface{}{}}, nil
	}

	seen := map[string]bool{}
	var unique []interface{}
	for _, item := range arr {
		key := fmt.Sprintf("%v", item)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}

	return map[string]interface{}{"items": unique, "removed": len(arr) - len(unique)}, nil
}

func execDateTime(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	operation, _ := config["operation"].(string)
	format, _ := config["format"].(string)
	value, _ := config["value"].(string)
	duration, _ := config["duration"].(string)

	if format == "" {
		format = time.RFC3339
	}

	switch operation {
	case "now":
		return map[string]interface{}{"result": time.Now().Format(format)}, nil
	case "format":
		value = resolveTemplate(value, execCtx)
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return map[string]interface{}{"result": value}, nil
		}
		return map[string]interface{}{"result": t.Format(format)}, nil
	case "parse":
		value = resolveTemplate(value, execCtx)
		t, err := time.Parse(format, value)
		if err != nil {
			return nil, fmt.Errorf("date_time: parse error: %w", err)
		}
		return map[string]interface{}{"result": t.Format(time.RFC3339)}, nil
	case "add":
		d, err := time.ParseDuration(duration)
		if err != nil {
			return nil, fmt.Errorf("date_time: invalid duration: %w", err)
		}
		return map[string]interface{}{"result": time.Now().Add(d).Format(format)}, nil
	default:
		return map[string]interface{}{"result": time.Now().Format(format)}, nil
	}
}

func execCrypto(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	operation, _ := config["operation"].(string)
	input, _ := config["input"].(string)
	input = resolveTemplate(input, execCtx)

	switch operation {
	case "md5":
		h := md5.Sum([]byte(input))
		return map[string]interface{}{"result": fmt.Sprintf("%x", h)}, nil
	case "sha256":
		h := sha256.Sum256([]byte(input))
		return map[string]interface{}{"result": fmt.Sprintf("%x", h)}, nil
	case "base64_encode":
		return map[string]interface{}{"result": base64.StdEncoding.EncodeToString([]byte(input))}, nil
	case "base64_decode":
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return nil, fmt.Errorf("crypto: base64 decode error: %w", err)
		}
		return map[string]interface{}{"result": string(decoded)}, nil
	default:
		return nil, fmt.Errorf("crypto: unknown operation: %s", operation)
	}
}

func execExtractJSON(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	jsonField, _ := config["json"].(string)
	path, _ := config["path"].(string)

	jsonStr := resolveTemplate(jsonField, execCtx)
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("extract_from_json: %w", err)
	}

	// Navigate path
	parts := strings.Split(path, ".")
	current := data
	for _, p := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[p]
		} else {
			return map[string]interface{}{"result": nil}, nil
		}
	}

	return map[string]interface{}{"result": current}, nil
}

// --- Integration Implementations ---

func execWebhookPost(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}, service string) (interface{}, error) {
	webhookURL, _ := config["webhookUrl"].(string)
	message, _ := config["message"].(string)

	if webhookURL == "" {
		return nil, fmt.Errorf("%s: webhookUrl is required", service)
	}

	message = resolveTemplate(message, execCtx)
	webhookURL = resolveTemplate(webhookURL, execCtx)

	payload := map[string]string{"text": message}
	if service == "discord" {
		payload = map[string]string{"content": message}
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", service, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", service, err)
	}
	defer resp.Body.Close()

	return map[string]interface{}{"status": resp.StatusCode, "service": service}, nil
}

func execTelegram(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	token, _ := config["botToken"].(string)
	chatID, _ := config["chatId"].(string)
	message, _ := config["message"].(string)

	if token == "" || chatID == "" {
		return nil, fmt.Errorf("telegram: botToken and chatId are required")
	}

	message = resolveTemplate(message, execCtx)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    message,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	defer resp.Body.Close()

	return map[string]interface{}{"status": resp.StatusCode}, nil
}

func execGitHub(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	token, _ := config["token"].(string)
	owner, _ := config["owner"].(string)
	repo, _ := config["repo"].(string)
	title, _ := config["title"].(string)
	body, _ := config["body"].(string)

	if token == "" || owner == "" || repo == "" {
		return nil, fmt.Errorf("github: token, owner, and repo are required")
	}

	title = resolveTemplate(title, execCtx)
	body = resolveTemplate(body, execCtx)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

// --- Additional Integration Implementations ---

func execGoogleSheets(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	token, _ := config["token"].(string)
	spreadsheetID, _ := config["spreadsheetId"].(string)
	sheetRange, _ := config["range"].(string)
	action, _ := config["action"].(string)

	if spreadsheetID == "" || sheetRange == "" {
		return nil, fmt.Errorf("google_sheets: spreadsheetId and range are required")
	}
	if token == "" {
		return nil, fmt.Errorf("google_sheets: token is required")
	}

	spreadsheetID = resolveTemplate(spreadsheetID, execCtx)
	sheetRange = resolveTemplate(sheetRange, execCtx)
	token = resolveTemplate(token, execCtx)

	baseURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s", spreadsheetID, sheetRange)

	var req *http.Request
	var err error

	switch action {
	case "append":
		values := config["values"]
		body, _ := json.Marshal(map[string]interface{}{"values": values})
		appendURL := baseURL + ":append?valueInputOption=RAW"
		req, err = http.NewRequestWithContext(ctx, "POST", appendURL, bytes.NewReader(body))
	default: // "read" or empty
		req, err = http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("google_sheets: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("google_sheets: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execNotion(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	token, _ := config["token"].(string)
	action, _ := config["action"].(string)
	databaseID, _ := config["databaseId"].(string)

	if token == "" {
		return nil, fmt.Errorf("notion: token is required")
	}

	token = resolveTemplate(token, execCtx)
	databaseID = resolveTemplate(databaseID, execCtx)

	var reqURL string
	var payload []byte

	switch action {
	case "create_page":
		reqURL = "https://api.notion.com/v1/pages"
		properties := config["properties"]
		body := map[string]interface{}{
			"parent":     map[string]string{"database_id": databaseID},
			"properties": properties,
		}
		payload, _ = json.Marshal(body)
	default: // "query_database" or empty
		if databaseID == "" {
			return nil, fmt.Errorf("notion: databaseId is required for query_database")
		}
		reqURL = fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", databaseID)
		filter := config["filter"]
		if filter != nil {
			payload, _ = json.Marshal(map[string]interface{}{"filter": filter})
		} else {
			payload = []byte("{}")
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("notion: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execStripe(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	apiKey, _ := config["apiKey"].(string)
	action, _ := config["action"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("stripe: apiKey is required")
	}

	apiKey = resolveTemplate(apiKey, execCtx)

	var reqURL string
	var formData string

	switch action {
	case "create_customer":
		reqURL = "https://api.stripe.com/v1/customers"
		email, _ := config["email"].(string)
		email = resolveTemplate(email, execCtx)
		formData = "email=" + email
	case "list_charges":
		reqURL = "https://api.stripe.com/v1/charges"
		formData = ""
	default: // "create_charge" or empty
		reqURL = "https://api.stripe.com/v1/charges"
		amount, _ := config["amount"].(float64)
		currency, _ := config["currency"].(string)
		if currency == "" {
			currency = "usd"
		}
		currency = resolveTemplate(currency, execCtx)
		formData = fmt.Sprintf("amount=%d&currency=%s&source=tok_visa", int(amount), currency)
	}

	var req *http.Request
	var err error

	if action == "list_charges" {
		req, err = http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(formData))
	}
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}

	req.SetBasicAuth(apiKey, "")
	if action != "list_charges" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execTwilioSMS(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	accountSID, _ := config["accountSid"].(string)
	authToken, _ := config["authToken"].(string)
	from, _ := config["from"].(string)
	to, _ := config["to"].(string)
	body, _ := config["body"].(string)

	if accountSID == "" || authToken == "" {
		return nil, fmt.Errorf("twilio_sms: accountSid and authToken are required")
	}
	if from == "" || to == "" || body == "" {
		return nil, fmt.Errorf("twilio_sms: from, to, and body are required")
	}

	accountSID = resolveTemplate(accountSID, execCtx)
	authToken = resolveTemplate(authToken, execCtx)
	from = resolveTemplate(from, execCtx)
	to = resolveTemplate(to, execCtx)
	body = resolveTemplate(body, execCtx)

	reqURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", accountSID)
	formData := fmt.Sprintf("From=%s&To=%s&Body=%s", from, to, body)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(formData))
	if err != nil {
		return nil, fmt.Errorf("twilio_sms: %w", err)
	}

	req.SetBasicAuth(accountSID, authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("twilio_sms: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execPostgresQuery(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	connURL, _ := config["connectionUrl"].(string)
	proxyURL, _ := config["proxyUrl"].(string)
	query, _ := config["query"].(string)

	if query == "" {
		return nil, fmt.Errorf("postgres_query: query is required")
	}
	query = resolveTemplate(query, execCtx)

	if connURL == "" && proxyURL == "" {
		return nil, fmt.Errorf("postgres_query: connectionUrl or proxyUrl is required")
	}

	// Use proxy URL if provided (for external Postgres via HTTP proxy)
	if connURL == "" {
		proxyURL = resolveTemplate(proxyURL, execCtx)
	} else {
		// For direct connection, use proxy approach via the connection URL
		// Since we don't ship a Postgres driver, route through an HTTP proxy
		proxyURL = connURL
	}

	payload, _ := json.Marshal(map[string]string{"query": query, "engine": "postgres"})
	req, err := http.NewRequestWithContext(ctx, "POST", proxyURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("postgres_query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if token, _ := config["token"].(string); token != "" {
		req.Header.Set("Authorization", "Bearer "+resolveTemplate(token, execCtx))
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("postgres_query: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execMySQLQuery(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	connURL, _ := config["connectionUrl"].(string)
	query, _ := config["query"].(string)

	if query == "" {
		return nil, fmt.Errorf("mysql_query: query is required")
	}
	query = resolveTemplate(query, execCtx)

	if connURL == "" {
		return nil, fmt.Errorf("mysql_query: connectionUrl is required")
	}
	connURL = resolveTemplate(connURL, execCtx)

	// Use database/sql with the MySQL driver (already imported in the project)
	db, err := sql.Open("mysql", connURL)
	if err != nil {
		return nil, fmt.Errorf("mysql_query: connect: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mysql_query: execute: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := map[string]interface{}{}
		for i, col := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[col] = v
		}
		results = append(results, row)
	}

	return map[string]interface{}{
		"rows":    results,
		"count":   len(results),
		"columns": cols,
	}, nil
}

func execRedisCommand(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	connURL, _ := config["connectionUrl"].(string)
	command, _ := config["command"].(string)

	if command == "" {
		return nil, fmt.Errorf("redis_command: command is required")
	}
	command = resolveTemplate(command, execCtx)

	if connURL == "" {
		connURL = "localhost:6379"
	}
	connURL = resolveTemplate(connURL, execCtx)

	// Connect via raw TCP and send RESP protocol
	conn, err := net.DialTimeout("tcp", connURL, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("redis_command: connect to %s: %w", connURL, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Parse command into parts (e.g., "GET mykey" → ["GET", "mykey"])
	parts := strings.Fields(command)
	// Build RESP array
	resp := fmt.Sprintf("*%d\r\n", len(parts))
	for _, p := range parts {
		resp += fmt.Sprintf("$%d\r\n%s\r\n", len(p), p)
	}
	conn.Write([]byte(resp))

	// Read response (simple: read up to 64KB)
	buf := make([]byte, 65536)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("redis_command: read: %w", err)
	}
	result := strings.TrimSpace(string(buf[:n]))

	// Parse RESP response
	var value interface{} = result
	if strings.HasPrefix(result, "+") {
		value = result[1:] // simple string
	} else if strings.HasPrefix(result, "-") {
		return nil, fmt.Errorf("redis_command: %s", result[1:]) // error
	} else if strings.HasPrefix(result, ":") {
		value = result[1:] // integer
	} else if strings.HasPrefix(result, "$") {
		// Bulk string — extract the data after the length line
		lines := strings.SplitN(result, "\r\n", 3)
		if len(lines) >= 2 {
			value = lines[1]
		}
	}

	return map[string]interface{}{
		"result":  value,
		"command": command,
	}, nil
}

func execS3(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	accessKey, _ := config["accessKeyId"].(string)
	secretKey, _ := config["secretAccessKey"].(string)
	region, _ := config["region"].(string)
	bucket, _ := config["bucket"].(string)
	key, _ := config["key"].(string)
	action, _ := config["action"].(string)

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("s3: accessKeyId and secretAccessKey are required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("s3: bucket is required")
	}

	accessKey = resolveTemplate(accessKey, execCtx)
	secretKey = resolveTemplate(secretKey, execCtx)
	region = resolveTemplate(region, execCtx)
	bucket = resolveTemplate(bucket, execCtx)
	key = resolveTemplate(key, execCtx)

	if region == "" {
		region = "us-east-1"
	}

	var method string
	var reqURL string
	var bodyReader io.Reader

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, region)

	switch action {
	case "put":
		method = "PUT"
		reqURL = fmt.Sprintf("https://%s/%s", host, key)
		data, _ := config["data"].(string)
		data = resolveTemplate(data, execCtx)
		bodyReader = strings.NewReader(data)
	case "list":
		method = "GET"
		reqURL = fmt.Sprintf("https://%s/?list-type=2", host)
		if key != "" {
			reqURL += "&prefix=" + key
		}
	default: // "get" or empty
		if key == "" {
			return nil, fmt.Errorf("s3: key is required for get")
		}
		method = "GET"
		reqURL = fmt.Sprintf("https://%s/%s", host, key)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}

	// Simplified AWS auth header
	now := time.Now().UTC()
	req.Header.Set("x-amz-date", now.Format("20060102T150405Z"))
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	cred := base64.StdEncoding.EncodeToString([]byte(accessKey + ":" + secretKey))
	req.Header.Set("Authorization", "AWS "+accessKey+":"+cred)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var result interface{}
	if json.Unmarshal(respBody, &result) != nil {
		result = string(respBody)
	}

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execSendGrid(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	apiKey, _ := config["apiKey"].(string)
	to, _ := config["to"].(string)
	from, _ := config["from"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("sendgrid: apiKey is required")
	}
	if to == "" || from == "" || subject == "" {
		return nil, fmt.Errorf("sendgrid: to, from, and subject are required")
	}

	apiKey = resolveTemplate(apiKey, execCtx)
	to = resolveTemplate(to, execCtx)
	from = resolveTemplate(from, execCtx)
	subject = resolveTemplate(subject, execCtx)
	body = resolveTemplate(body, execCtx)

	payload, _ := json.Marshal(map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{{"email": to}},
			},
		},
		"from":    map[string]string{"email": from},
		"subject": subject,
		"content": []map[string]string{
			{"type": "text/html", "value": body},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("sendgrid: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("sendgrid: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

func execJira(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	domain, _ := config["domain"].(string)
	email, _ := config["email"].(string)
	apiToken, _ := config["apiToken"].(string)
	action, _ := config["action"].(string)
	projectKey, _ := config["projectKey"].(string)
	summary, _ := config["summary"].(string)
	description, _ := config["description"].(string)

	if domain == "" || email == "" || apiToken == "" {
		return nil, fmt.Errorf("jira: domain, email, and apiToken are required")
	}

	domain = resolveTemplate(domain, execCtx)
	email = resolveTemplate(email, execCtx)
	apiToken = resolveTemplate(apiToken, execCtx)
	projectKey = resolveTemplate(projectKey, execCtx)
	summary = resolveTemplate(summary, execCtx)
	description = resolveTemplate(description, execCtx)

	baseURL := fmt.Sprintf("https://%s.atlassian.net/rest/api/3", domain)

	var req *http.Request
	var err error

	switch action {
	case "list_issues":
		jql := fmt.Sprintf("project=%s", projectKey)
		reqURL := fmt.Sprintf("%s/search?jql=%s", baseURL, jql)
		req, err = http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	default: // "create_issue" or empty
		if projectKey == "" || summary == "" {
			return nil, fmt.Errorf("jira: projectKey and summary are required for create_issue")
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"fields": map[string]interface{}{
				"project":     map[string]string{"key": projectKey},
				"summary":     summary,
				"description": map[string]interface{}{
					"type":    "doc",
					"version": 1,
					"content": []map[string]interface{}{
						{
							"type": "paragraph",
							"content": []map[string]interface{}{
								{"type": "text", "text": description},
							},
						},
					},
				},
				"issuetype": map[string]string{"name": "Task"},
			},
		})
		req, err = http.NewRequestWithContext(ctx, "POST", baseURL+"/issue", bytes.NewReader(payload))
	}
	if err != nil {
		return nil, fmt.Errorf("jira: %w", err)
	}

	req.SetBasicAuth(email, apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{"status": resp.StatusCode, "response": result}, nil
}

// --- Try/Catch and Stop Implementations ---

// execTryCatch implements the try_catch node type.
// Config: tryNodes (array of node IDs to try), catchTarget (node ID for error handling).
// It returns skipTargets: on success, catchTarget is skipped; on implied failure, tryNodes are skipped
// and catchTarget is executed. The actual try/catch wrapping of execution is handled in RunWorkflow
// via the tryCatch tracking; here we just set up the skip targets for the success path.
func execTryCatch(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, []string, error) {
	catchTarget, _ := config["catchTarget"].(string)

	tryNodeIDs := extractStringSlice(config, "tryNodes")

	// By default, assume success: skip the catch branch
	var skip []string
	if catchTarget != "" {
		skip = append(skip, catchTarget)
	}

	return map[string]interface{}{
		"tryNodes":    tryNodeIDs,
		"catchTarget": catchTarget,
		"branch":      "success",
	}, skip, nil
}

// execStopAndError immediately fails the workflow with a configurable error message.
func execStopAndError(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	message, _ := config["message"].(string)
	if message == "" {
		message = "Workflow stopped by stop_and_error node"
	}
	message = resolveTemplate(message, execCtx)
	return map[string]interface{}{"message": message}, fmt.Errorf("stop_and_error: %s", message)
}

// extractStringSlice reads a string slice from a config map value ([]interface{} of strings).
func extractStringSlice(config map[string]interface{}, key string) []string {
	raw, ok := config[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// --- JavaScript-like Code Execution ---

func execJavaScript(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	code, _ := config["code"].(string)
	if code == "" {
		return nil, fmt.Errorf("javascript: code is required")
	}

	vm := goja.New()
	// Set execution timeout (5 seconds)
	timer := time.AfterFunc(5*time.Second, func() { vm.Interrupt("execution timeout (5s)") })
	defer timer.Stop()

	// Inject execution context as $input
	vm.Set("$input", execCtx)
	vm.Set("$json", execCtx)
	vm.Set("$env", func(key string) string { return os.Getenv(key) })
	vm.Set("$now", time.Now().UTC().Format(time.RFC3339))

	// Inject console.log
	logOutput := &strings.Builder{}
	console := map[string]interface{}{
		"log": func(args ...interface{}) {
			for i, a := range args {
				if i > 0 {
					logOutput.WriteString(" ")
				}
				fmt.Fprint(logOutput, a)
			}
			logOutput.WriteString("\n")
		},
	}
	vm.Set("console", console)

	// Inject JSON helpers
	vm.Set("JSON", map[string]interface{}{
		"stringify": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"parse": func(s string) interface{} {
			var v interface{}
			json.Unmarshal([]byte(s), &v)
			return v
		},
	})

	// Execute
	val, err := vm.RunString(code)
	if err != nil {
		return nil, fmt.Errorf("javascript: %w", err)
	}

	// Export result
	var result interface{}
	if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
		result = val.Export()
	}

	return map[string]interface{}{
		"result": result,
		"logs":   logOutput.String(),
	}, nil
}

// --- AI Node Implementations ---

func execAITransform(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	model, _ := config["model"].(string)
	prompt, _ := config["prompt"].(string)
	apiKey, _ := config["apiKey"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("ai_transform: apiKey is required")
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	prompt = resolveTemplate(prompt, execCtx)

	// Detect provider from model name
	apiURL := "https://api.anthropic.com/v1/messages"
	var payload []byte
	var req *http.Request

	if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") {
		// OpenAI
		apiURL = "https://api.openai.com/v1/chat/completions"
		payload, _ = json.Marshal(map[string]interface{}{
			"model":      model,
			"messages":   []map[string]string{{"role": "user", "content": prompt}},
			"max_tokens": 2048,
		})
		req, _ = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		// Anthropic (default)
		payload, _ = json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": 2048,
			"messages":   []map[string]string{{"role": "user", "content": prompt}},
		})
		req, _ = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payload))
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai_transform: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Extract text from response
	text := ""
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			text, _ = block["text"].(string)
		}
	} else if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				text, _ = msg["content"].(string)
			}
		}
	}

	return map[string]interface{}{
		"result":   text,
		"model":    model,
		"raw":      result,
	}, nil
}

func execAISummarize(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	model, _ := config["model"].(string)
	text, _ := config["text"].(string)
	maxLength, _ := config["maxLength"].(string)
	apiKey, _ := config["apiKey"].(string)

	if maxLength == "" {
		maxLength = "200"
	}
	text = resolveTemplate(text, execCtx)
	prompt := fmt.Sprintf("Summarize the following text in under %s words:\n\n%s", maxLength, text)

	// Reuse ai_transform with constructed prompt
	return execAITransform(ctx, map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"apiKey": apiKey,
	}, execCtx)
}

func execAIAgent(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	model, _ := config["model"].(string)
	systemPrompt, _ := config["systemPrompt"].(string)
	userMessage, _ := config["userMessage"].(string)
	apiKey, _ := config["apiKey"].(string)
	maxStepsStr, _ := config["maxSteps"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("ai_agent: apiKey is required")
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxSteps := 5
	if n, err := strconv.Atoi(maxStepsStr); err == nil && n > 0 {
		maxSteps = n
	}
	if maxSteps > 20 {
		maxSteps = 20
	}

	systemPrompt = resolveTemplate(systemPrompt, execCtx)
	userMessage = resolveTemplate(userMessage, execCtx)

	messages := []map[string]string{{"role": "user", "content": userMessage}}
	var allResults []interface{}

	for step := 0; step < maxSteps; step++ {
		payload, _ := json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": 2048,
			"system":     systemPrompt,
			"messages":   messages,
		})

		req, _ := http.NewRequestWithContext(ctx, "POST",
			"https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
		if err != nil {
			return nil, fmt.Errorf("ai_agent step %d: %w", step, err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		text := ""
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if block, ok := content[0].(map[string]interface{}); ok {
				text, _ = block["text"].(string)
			}
		}

		allResults = append(allResults, map[string]interface{}{
			"step": step + 1, "text": text,
		})

		// Check if agent signaled completion (contains "DONE" or "FINAL ANSWER")
		if strings.Contains(strings.ToUpper(text), "FINAL ANSWER") ||
			strings.Contains(strings.ToUpper(text), "DONE") ||
			result["stop_reason"] == "end_turn" {
			break
		}

		// Add assistant response and continue
		messages = append(messages, map[string]string{"role": "assistant", "content": text})
		messages = append(messages, map[string]string{"role": "user", "content": "Continue."})
	}

	finalText := ""
	if len(allResults) > 0 {
		last := allResults[len(allResults)-1].(map[string]interface{})
		finalText, _ = last["text"].(string)
	}

	return map[string]interface{}{
		"result": finalText,
		"steps":  allResults,
		"model":  model,
	}, nil
}

// --- Applad-native Node Implementations ---

// ApplAdBaseURL is the internal API URL for Applad-native nodes.
var ApplAdBaseURL = "http://localhost:8080/v1"

func execApplad(ctx context.Context, config map[string]interface{}, execCtx map[string]interface{}, service string) (interface{}, error) {
	action, _ := config["action"].(string)
	projectID, _ := config["projectId"].(string)
	apiKey, _ := config["apiKey"].(string)

	// Auto-inject project from workflow context if not specified
	if projectID == "" {
		if wfCtx, ok := execCtx["__projectId__"].(string); ok {
			projectID = wfCtx
		}
	}

	if action == "" {
		return nil, fmt.Errorf("applad_%s: action is required", service)
	}

	var method, path string
	var body interface{}

	switch service {
	case "auth":
		switch action {
		case "create_user":
			method, path = "POST", "/users"
			body = map[string]interface{}{
				"email":    resolveTemplate(getStr(config, "email"), execCtx),
				"password": resolveTemplate(getStr(config, "password"), execCtx),
				"name":     resolveTemplate(getStr(config, "name"), execCtx),
			}
		case "get_user":
			method, path = "GET", "/users/"+resolveTemplate(getStr(config, "userId"), execCtx)
		case "list_users":
			method, path = "GET", "/users"
		case "update_user":
			method, path = "PATCH", "/users/"+resolveTemplate(getStr(config, "userId"), execCtx)
			body = config["data"]
		case "delete_user":
			method, path = "DELETE", "/users/"+resolveTemplate(getStr(config, "userId"), execCtx)
		default:
			return nil, fmt.Errorf("applad_auth: unknown action: %s", action)
		}

	case "database":
		dbID := resolveTemplate(getStr(config, "databaseId"), execCtx)
		collID := resolveTemplate(getStr(config, "collectionId"), execCtx)
		docID := resolveTemplate(getStr(config, "documentId"), execCtx)
		basePath := fmt.Sprintf("/databases/%s/tables/%s", dbID, collID)

		switch action {
		case "create_document":
			method, path = "POST", basePath+"/rows"
			body = config["data"]
		case "get_document":
			method, path = "GET", basePath+"/rows/"+docID
		case "list_documents":
			method, path = "GET", basePath+"/rows"
		case "update_document":
			method, path = "PATCH", basePath+"/rows/"+docID
			body = config["data"]
		case "delete_document":
			method, path = "DELETE", basePath+"/rows/"+docID
		case "upsert_document":
			method, path = "POST", basePath+"/rows"
			body = config["data"]
		default:
			return nil, fmt.Errorf("applad_database: unknown action: %s", action)
		}

	case "storage":
		bucketID := resolveTemplate(getStr(config, "bucketId"), execCtx)
		fileID := resolveTemplate(getStr(config, "fileId"), execCtx)

		switch action {
		case "list_files":
			method, path = "GET", fmt.Sprintf("/storage/buckets/%s/files", bucketID)
		case "get_file":
			method, path = "GET", fmt.Sprintf("/storage/buckets/%s/files/%s", bucketID, fileID)
		case "delete_file":
			method, path = "DELETE", fmt.Sprintf("/storage/buckets/%s/files/%s", bucketID, fileID)
		default:
			return nil, fmt.Errorf("applad_storage: unknown action: %s", action)
		}

	case "functions":
		targetID := resolveTemplate(getStr(config, "targetId"), execCtx)
		switch action {
		case "invoke":
			method, path = "POST", fmt.Sprintf("/deploy/targets/%s/executions", targetID)
			body = config["data"]
		case "list_executions":
			method, path = "GET", fmt.Sprintf("/deploy/targets/%s/executions", targetID)
		default:
			return nil, fmt.Errorf("applad_functions: unknown action: %s", action)
		}

	case "messaging":
		switch action {
		case "send_email":
			method, path = "POST", "/messaging/messages/email"
			body = map[string]interface{}{
				"to":      resolveTemplate(getStr(config, "to"), execCtx),
				"subject": resolveTemplate(getStr(config, "subject"), execCtx),
				"body":    resolveTemplate(getStr(config, "body"), execCtx),
			}
		case "send_sms":
			method, path = "POST", "/messaging/messages/sms"
			body = map[string]interface{}{
				"to":   resolveTemplate(getStr(config, "to"), execCtx),
				"body": resolveTemplate(getStr(config, "body"), execCtx),
			}
		case "send_push":
			method, path = "POST", "/messaging/messages/push"
			body = map[string]interface{}{
				"title": resolveTemplate(getStr(config, "title"), execCtx),
				"body":  resolveTemplate(getStr(config, "body"), execCtx),
			}
		default:
			return nil, fmt.Errorf("applad_messaging: unknown action: %s", action)
		}
	}

	url := ApplAdBaseURL + path
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Applad-Project", projectID)
	if apiKey != "" {
		req.Header.Set("X-Applad-Key", apiKey)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("applad_%s: %w", service, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var result interface{}
	json.Unmarshal(respBody, &result)

	return map[string]interface{}{
		"statusCode": resp.StatusCode,
		"body":       result,
	}, nil
}

func getStr(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

// --- Additional Flow Node Implementations ---

func execSort(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field, _ := config["field"].(string)
	order, _ := config["order"].(string)
	itemsField, _ := config["items"].(string)

	items := resolveFieldValue(itemsField, execCtx)
	arr, ok := items.([]interface{})
	if !ok {
		return map[string]interface{}{"items": []interface{}{}}, nil
	}

	sorted := make([]interface{}, len(arr))
	copy(sorted, arr)

	// Sort by field value
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			vi := fmt.Sprintf("%v", extractField(sorted[i], field))
			vj := fmt.Sprintf("%v", extractField(sorted[j], field))
			if (order == "desc" && vi < vj) || (order != "desc" && vi > vj) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return map[string]interface{}{"items": sorted}, nil
}

func extractField(item interface{}, field string) interface{} {
	if m, ok := item.(map[string]interface{}); ok && field != "" {
		return m[field]
	}
	return item
}

func execRenameKeys(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	mapping, ok := config["mapping"].(map[string]interface{})
	if !ok {
		mappingStr, _ := config["mapping"].(string)
		if mappingStr != "" {
			json.Unmarshal([]byte(resolveTemplate(mappingStr, execCtx)), &mapping)
		}
	}
	if mapping == nil {
		return map[string]interface{}{}, nil
	}

	result := cloneMap(execCtx)
	for oldKey, newKey := range mapping {
		newKeyStr, _ := newKey.(string)
		if newKeyStr != "" {
			if val, exists := result[oldKey]; exists {
				result[newKeyStr] = val
				delete(result, oldKey)
			}
		}
	}

	return result, nil
}

func execCompareDatasets(config map[string]interface{}, execCtx map[string]interface{}) (interface{}, error) {
	field1, _ := config["input1"].(string)
	field2, _ := config["input2"].(string)
	keyField, _ := config["keyField"].(string)

	data1 := resolveFieldValue(field1, execCtx)
	data2 := resolveFieldValue(field2, execCtx)

	arr1, _ := data1.([]interface{})
	arr2, _ := data2.([]interface{})

	set1 := map[string]interface{}{}
	set2 := map[string]interface{}{}

	for _, item := range arr1 {
		key := fmt.Sprintf("%v", extractField(item, keyField))
		set1[key] = item
	}
	for _, item := range arr2 {
		key := fmt.Sprintf("%v", extractField(item, keyField))
		set2[key] = item
	}

	var added, removed, unchanged []interface{}

	for key, item := range set2 {
		if _, exists := set1[key]; !exists {
			added = append(added, item)
		} else {
			unchanged = append(unchanged, item)
		}
	}
	for key, item := range set1 {
		if _, exists := set2[key]; !exists {
			removed = append(removed, item)
		}
	}

	return map[string]interface{}{
		"added":     added,
		"removed":   removed,
		"unchanged": unchanged,
		"addedCount":   len(added),
		"removedCount": len(removed),
	}, nil
}

// --- Helpers ---

// topoSort returns node IDs in topological order.
func topoSort(nodes []Node, edges []Edge) []string {
	inDegree := map[string]int{}
	adj := map[string][]string{}

	for _, n := range nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		inDegree[e.Target]++
	}

	var queue []string
	for _, n := range nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var order []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// If there are nodes not in the order (cycle or disconnected), append them
	seen := map[string]bool{}
	for _, id := range order {
		seen[id] = true
	}
	for _, n := range nodes {
		if !seen[n.ID] {
			order = append(order, n.ID)
		}
	}

	return order
}

// resolveTemplate evaluates a Go text/template string against the execution context.
func resolveTemplate(tmplStr string, data map[string]interface{}) string {
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr
	}
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return tmplStr
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return tmplStr
	}
	return buf.String()
}

// resolveFieldValue navigates a dot-separated path in the execution context.
func resolveFieldValue(field string, data map[string]interface{}) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

// evaluateCondition compares two values with the given operator.
func evaluateCondition(actual interface{}, operator string, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	switch operator {
	case "eq", "==", "equals":
		return actualStr == expectedStr
	case "neq", "!=", "not_equals":
		return actualStr != expectedStr
	case "contains":
		return strings.Contains(actualStr, expectedStr)
	case "not_contains":
		return !strings.Contains(actualStr, expectedStr)
	case "starts_with":
		return strings.HasPrefix(actualStr, expectedStr)
	case "ends_with":
		return strings.HasSuffix(actualStr, expectedStr)
	case "empty":
		return actualStr == "" || actualStr == "<nil>"
	case "not_empty":
		return actualStr != "" && actualStr != "<nil>"
	default:
		return actualStr == expectedStr
	}
}

func flattenHeaders(h http.Header) map[string]string {
	flat := make(map[string]string, len(h))
	for k, v := range h {
		flat[k] = strings.Join(v, ", ")
	}
	return flat
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
