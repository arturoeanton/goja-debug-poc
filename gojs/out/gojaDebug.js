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
const Net = __importStar(require("net"));
class GojaDebugSession extends vscode_debugadapter_1.LoggingDebugSession {
    constructor() {
        super("goja-debug.txt");
        this._variableHandles = new vscode_debugadapter_1.Handles();
        this._configurationDone = new Subject();
        this._debugServerPort = 0;
        this._messageBuffer = '';
        this._seq = 1;
        this._dapPendingRequests = new Map();
        this._isAttach = false;
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
        // Send configurationDone to DAP server
        this.sendDAPRequest('configurationDone', {}, (dapResponse) => {
            this._configurationDone.notify();
        });
        this.sendResponse(response);
    }
    async launchRequest(response, args) {
        this._isAttach = false;
        this._program = args.program;
        const stopOnEntry = args.stopOnEntry || false;
        this._debugServerPort = args.debugServer || 5678;
        // Connect to debug server
        await this.connectToDebugServer();
        // Send initialize to DAP server
        await this.sendDAPRequestAsync('initialize', {
            clientID: 'vscode',
            clientName: 'Visual Studio Code',
            adapterID: 'goja',
            pathFormat: 'path',
            linesStartAt1: true,
            columnsStartAt1: true
        });
        // Send launch to DAP server
        await this.sendDAPRequestAsync('launch', {
            request: 'launch',
            program: this._program,
            stopOnEntry: stopOnEntry
        });
        await this._configurationDone.wait(1000);
        this.sendResponse(response);
    }
    async attachRequest(response, args) {
        this._isAttach = true;
        this._debugServerPort = args.debugServer || 5678;
        await this.connectToDebugServer();
        // Send initialize to DAP server
        await this.sendDAPRequestAsync('initialize', {
            clientID: 'vscode',
            clientName: 'Visual Studio Code',
            adapterID: 'goja',
            pathFormat: 'path',
            linesStartAt1: true,
            columnsStartAt1: true
        });
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
            this._debugServerClient.on('data', (data) => {
                this.handleDAPData(data);
            });
        });
    }
    handleDAPData(data) {
        this._messageBuffer += data.toString();
        while (true) {
            const headerEnd = this._messageBuffer.indexOf('\r\n\r\n');
            if (headerEnd === -1) {
                break;
            }
            const header = this._messageBuffer.substring(0, headerEnd);
            const contentLengthMatch = header.match(/Content-Length: (\d+)/);
            if (!contentLengthMatch) {
                this._messageBuffer = this._messageBuffer.substring(headerEnd + 4);
                continue;
            }
            const contentLength = parseInt(contentLengthMatch[1], 10);
            const messageStart = headerEnd + 4;
            if (this._messageBuffer.length < messageStart + contentLength) {
                break;
            }
            const message = this._messageBuffer.substring(messageStart, messageStart + contentLength);
            this._messageBuffer = this._messageBuffer.substring(messageStart + contentLength);
            try {
                const dapMessage = JSON.parse(message);
                this.handleDAPMessage(dapMessage);
            }
            catch (e) {
                console.error('Failed to parse DAP message:', e);
            }
        }
    }
    handleDAPMessage(message) {
        if (message.type === 'response') {
            const handler = this._dapPendingRequests.get(message.request_seq);
            if (handler) {
                this._dapPendingRequests.delete(message.request_seq);
                handler(message);
            }
        }
        else if (message.type === 'event') {
            switch (message.event) {
                case 'stopped':
                    const stoppedBody = message.body;
                    this.sendEvent(new vscode_debugadapter_1.StoppedEvent(stoppedBody.reason || 'breakpoint', stoppedBody.threadId || GojaDebugSession.threadID));
                    break;
                case 'output':
                    const outputBody = message.body;
                    this.sendEvent(new vscode_debugadapter_1.OutputEvent(outputBody.output, outputBody.category));
                    break;
                case 'terminated':
                    this.sendEvent(new vscode_debugadapter_1.TerminatedEvent());
                    break;
            }
        }
    }
    sendDAPRequest(command, args, handler) {
        const request = {
            seq: this._seq++,
            type: 'request',
            command: command,
            arguments: args
        };
        if (handler) {
            this._dapPendingRequests.set(request.seq, handler);
        }
        const json = JSON.stringify(request);
        const message = `Content-Length: ${Buffer.byteLength(json)}\r\n\r\n${json}`;
        this._debugServerClient?.write(message);
    }
    sendDAPRequestAsync(command, args) {
        return new Promise((resolve) => {
            this.sendDAPRequest(command, args, resolve);
        });
    }
    async setBreakPointsRequest(response, args) {
        const path = args.source.path;
        const clientBreakpoints = args.breakpoints || [];
        const clientLines = args.lines || [];
        // Send to DAP server
        const dapResponse = await this.sendDAPRequestAsync('setBreakpoints', {
            source: { path: path },
            breakpoints: clientBreakpoints.length > 0
                ? clientBreakpoints
                : clientLines.map(line => ({ line }))
        });
        const breakpoints = [];
        if (dapResponse.body?.breakpoints) {
            dapResponse.body.breakpoints.forEach((bp) => {
                breakpoints.push({
                    verified: bp.verified,
                    line: bp.line,
                    source: args.source
                });
            });
        }
        response.body = {
            breakpoints: breakpoints
        };
        this.sendResponse(response);
    }
    threadsRequest(response) {
        this.sendDAPRequest('threads', {}, (dapResponse) => {
            if (dapResponse.body?.threads) {
                response.body = {
                    threads: dapResponse.body.threads
                };
            }
            else {
                response.body = {
                    threads: [new vscode_debugadapter_1.Thread(GojaDebugSession.threadID, "main")]
                };
            }
            this.sendResponse(response);
        });
    }
    stackTraceRequest(response, args) {
        this.sendDAPRequest('stackTrace', {
            threadId: args.threadId,
            startFrame: args.startFrame,
            levels: args.levels
        }, (dapResponse) => {
            if (dapResponse.body) {
                response.body = dapResponse.body;
            }
            else {
                response.body = {
                    stackFrames: [],
                    totalFrames: 0
                };
            }
            this.sendResponse(response);
        });
    }
    scopesRequest(response, args) {
        this.sendDAPRequest('scopes', {
            frameId: args.frameId
        }, (dapResponse) => {
            if (dapResponse.body?.scopes) {
                response.body = {
                    scopes: dapResponse.body.scopes
                };
            }
            else {
                response.body = {
                    scopes: []
                };
            }
            this.sendResponse(response);
        });
    }
    variablesRequest(response, args) {
        this.sendDAPRequest('variables', {
            variablesReference: args.variablesReference,
            filter: args.filter,
            start: args.start,
            count: args.count
        }, (dapResponse) => {
            if (dapResponse.body?.variables) {
                response.body = {
                    variables: dapResponse.body.variables
                };
            }
            else {
                response.body = {
                    variables: []
                };
            }
            this.sendResponse(response);
        });
    }
    continueRequest(response, args) {
        this.sendDAPRequest('continue', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }
    nextRequest(response, args) {
        this.sendDAPRequest('next', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }
    stepInRequest(response, args) {
        this.sendDAPRequest('stepIn', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }
    stepOutRequest(response, args) {
        this.sendDAPRequest('stepOut', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }
    evaluateRequest(response, args) {
        this.sendDAPRequest('evaluate', {
            expression: args.expression,
            frameId: args.frameId,
            context: args.context
        }, (dapResponse) => {
            if (dapResponse.body) {
                response.body = dapResponse.body;
            }
            else {
                response.body = {
                    result: 'Error',
                    variablesReference: 0
                };
            }
            this.sendResponse(response);
        });
    }
    pauseRequest(response, args) {
        this.sendDAPRequest('pause', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }
    disconnectRequest(response, args) {
        this.sendDAPRequest('disconnect', {
            restart: args.restart,
            terminateDebuggee: args.terminateDebuggee
        }, () => {
            if (this._debugServerClient) {
                this._debugServerClient.destroy();
            }
            if (this._gojaProcess) {
                this._gojaProcess.kill();
            }
            this.sendResponse(response);
        });
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