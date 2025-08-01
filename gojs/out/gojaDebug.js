"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GojaDebugSession = void 0;
const vscode_debugadapter_1 = require("vscode-debugadapter");
const path = __importStar(require("path"));
const child_process_1 = require("child_process");
const Net = __importStar(require("net"));
class GojaDebugSession extends vscode_debugadapter_1.LoggingDebugSession {
    constructor() {
        super("goja-debug.txt");
        this._variableHandles = new vscode_debugadapter_1.Handles();
        this._configurationDone = new Subject();
        this._debugServerPort = 0;
        this.setDebuggerLinesStartAt1(true);
        this.setDebuggerColumnsStartAt1(true);
    }
    initializeRequest(response, args) {
        response.body = response.body || {};
        response.body.supportsConfigurationDoneRequest = true;
        response.body.supportsEvaluateForHovers = true;
        response.body.supportsStepBack = false;
        response.body.supportsSetVariable = false;
        response.body.supportsRestartFrame = false;
        response.body.supportsStepInTargetsRequest = false;
        response.body.supportsGotoTargetsRequest = false;
        response.body.supportsCompletionsRequest = false;
        response.body.supportsTerminateRequest = true;
        this.sendResponse(response);
        this.sendEvent(new vscode_debugadapter_1.InitializedEvent());
    }
    configurationDoneRequest(response, args) {
        super.configurationDoneRequest(response, args);
        this._configurationDone.notify();
    }
    async launchRequest(response, args) {
        await this._configurationDone.wait(1000);
        const program = args.program;
        const stopOnEntry = args.stopOnEntry || false;
        this._debugServerPort = args.debugServer || 5678;
        // Start gojs with debug flag
        const gojsPath = path.join(__dirname, '../../dap/gojs');
        const gojsArgs = ['-d', '-port', this._debugServerPort.toString(), '-f', program];
        this._gojaProcess = (0, child_process_1.spawn)('go', ['run', gojsPath + '.go', ...gojsArgs], {
            cwd: path.dirname(program)
        });
        this._gojaProcess.stdout?.on('data', (data) => {
            this.sendEvent(new vscode_debugadapter_1.OutputEvent(data.toString(), 'stdout'));
        });
        this._gojaProcess.stderr?.on('data', (data) => {
            this.sendEvent(new vscode_debugadapter_1.OutputEvent(data.toString(), 'stderr'));
        });
        this._gojaProcess.on('exit', (code) => {
            this.sendEvent(new vscode_debugadapter_1.TerminatedEvent());
        });
        // Give the debug server time to start
        await new Promise(resolve => setTimeout(resolve, 1000));
        // Connect to debug server
        await this.connectToDebugServer();
        this.sendResponse(response);
    }
    async attachRequest(response, args) {
        this._debugServerPort = args.debugServer || 5678;
        await this.connectToDebugServer();
        this.sendResponse(response);
    }
    async connectToDebugServer() {
        return new Promise((resolve, reject) => {
            this._debugServerClient = Net.connect(this._debugServerPort, 'localhost');
            this._debugServerClient.on('connect', () => {
                resolve();
            });
            this._debugServerClient.on('error', (err) => {
                reject(err);
            });
            this._debugServerClient.on('close', () => {
                this.sendEvent(new vscode_debugadapter_1.TerminatedEvent());
            });
        });
    }
    setBreakPointsRequest(response, args) {
        const path = args.source.path;
        const clientLines = args.lines || [];
        const breakpoints = clientLines.map(l => {
            const bp = {
                verified: true,
                line: l,
                source: args.source
            };
            return bp;
        });
        response.body = {
            breakpoints: breakpoints
        };
        this.sendResponse(response);
    }
    threadsRequest(response) {
        response.body = {
            threads: [
                new vscode_debugadapter_1.Thread(GojaDebugSession.threadID, "main")
            ]
        };
        this.sendResponse(response);
    }
    stackTraceRequest(response, args) {
        const startFrame = typeof args.startFrame === 'number' ? args.startFrame : 0;
        const maxLevels = typeof args.levels === 'number' ? args.levels : 1000;
        const frames = [];
        // This would be populated from the debug adapter
        response.body = {
            stackFrames: frames,
            totalFrames: frames.length
        };
        this.sendResponse(response);
    }
    scopesRequest(response, args) {
        const scopes = [];
        scopes.push(new vscode_debugadapter_1.Scope("Local", this._variableHandles.create("local"), false));
        response.body = {
            scopes: scopes
        };
        this.sendResponse(response);
    }
    variablesRequest(response, args) {
        const variables = [];
        // This would be populated from the debug adapter
        response.body = {
            variables: variables
        };
        this.sendResponse(response);
    }
    continueRequest(response, args) {
        this.sendResponse(response);
    }
    nextRequest(response, args) {
        this.sendResponse(response);
    }
    stepInRequest(response, args) {
        this.sendResponse(response);
    }
    stepOutRequest(response, args) {
        this.sendResponse(response);
    }
    evaluateRequest(response, args) {
        response.body = {
            result: 'Not implemented',
            variablesReference: 0
        };
        this.sendResponse(response);
    }
    pauseRequest(response, args) {
        this.sendResponse(response);
    }
    disconnectRequest(response, args) {
        if (this._debugServerClient) {
            this._debugServerClient.destroy();
        }
        if (this._gojaProcess) {
            this._gojaProcess.kill();
        }
        this.sendResponse(response);
    }
}
exports.GojaDebugSession = GojaDebugSession;
GojaDebugSession.threadID = 1;
class Subject {
    constructor() {
        this._callbacks = [];
    }
    notify() {
        this._callbacks.forEach(cb => cb());
        this._callbacks = [];
    }
    wait(timeout) {
        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                reject(new Error('timeout'));
            }, timeout);
            this._callbacks.push(() => {
                clearTimeout(timer);
                resolve();
            });
        });
    }
}
//# sourceMappingURL=gojaDebug.js.map