"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ApplAdServer = void 0;
const analytics_1 = require("./analytics");
const users_1 = require("./users");
const databases_1 = require("./databases");
const storage_1 = require("./storage");
const functions_1 = require("./functions");
const teams_1 = require("./teams");
const workflows_1 = require("./workflows");
const messaging_1 = require("./messaging");
const deploy_1 = require("./deploy");
const edge_1 = require("./edge");
const flags_1 = require("./flags");
const regions_1 = require("./regions");
const search_1 = require("./search");
const vectors_1 = require("./vectors");
const errors_1 = require("./errors");
class ApplAdServer {
    constructor(config) {
        this.endpoint = config.endpoint.replace(/\/$/, '');
        this.projectId = config.projectId;
        this.headers = {
            'X-Applad-Project': config.projectId,
            'X-Applad-Key': config.apiKey,
            'Content-Type': 'application/json',
        };
        this.analytics = new analytics_1.Analytics(this);
        this.users = new users_1.Users(this);
        this.databases = new databases_1.Databases(this);
        this.storage = new storage_1.Storage(this);
        this.functions = new functions_1.Functions(this);
        this.teams = new teams_1.Teams(this);
        this.workflows = new workflows_1.Workflows(this);
        this.messaging = new messaging_1.Messaging(this);
        this.deploy = new deploy_1.Deploy(this);
        this.edge = new edge_1.Edge(this);
        this.flags = new flags_1.Flags(this);
        this.regions = new regions_1.Regions(this);
        this.search = new search_1.Search(this);
        this.vectors = new vectors_1.Vectors(this);
    }
    async call(method, path, body) {
        const res = await fetch(`${this.endpoint}/v1${path}`, {
            method,
            headers: this.headers,
            body: body ? JSON.stringify(body) : undefined,
        });
        if (!res.ok)
            throw await (0, errors_1.errorFromResponse)(res, method, path);
        if (res.status === 204)
            return undefined;
        return res.json();
    }
    async upload(path, formData) {
        const headers = { ...this.headers };
        delete headers['Content-Type'];
        const res = await fetch(`${this.endpoint}/v1${path}`, {
            method: 'POST',
            headers,
            body: formData,
        });
        if (!res.ok)
            throw await (0, errors_1.errorFromResponse)(res, 'POST', path);
        return res.json();
    }
    async download(path) {
        const res = await fetch(`${this.endpoint}/v1${path}`, {
            headers: this.headers,
        });
        if (!res.ok)
            throw await (0, errors_1.errorFromResponse)(res, 'GET', path);
        return res.arrayBuffer();
    }
}
exports.ApplAdServer = ApplAdServer;
