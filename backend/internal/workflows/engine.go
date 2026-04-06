package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"
	"time"
)

// RunWorkflow executes a workflow and returns step logs.
func RunWorkflow(ctx context.Context, wf *Workflow, triggerData map[string]interface{}) ([]StepLog, error) {
	if len(wf.Nodes) == 0 {
		return []StepLog{}, nil
	}

	// Build node lookup and adjacency list
	nodeMap := make(map[string]*Node, len(wf.Nodes))
	for i := range wf.Nodes {
		nodeMap[wf.Nodes[i].ID] = &wf.Nodes[i]
	}

	order := topoSort(wf.Nodes, wf.Edges)

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
		output, skipTargets, err := executeNode(ctx, node, execCtx)
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
	case "if_condition":
		output, skipTargets, err = execIfCondition(node.Config, execCtx)
	case "delay":
		output, err = execDelay(ctx, node.Config)
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
