import {
    LoggingDebugSession,
    InitializedEvent,
    TerminatedEvent,
    StoppedEvent,
    BreakpointEvent,
    OutputEvent,
    Thread,
    StackFrame,
    Scope,
    Source,
    Handles,
    Breakpoint
} from 'vscode-debugadapter';
import { DebugProtocol } from 'vscode-debugprotocol';
import * as path from 'path';
import * as fs from 'fs';
import { spawn, ChildProcess } from 'child_process';
import * as Net from 'net';

interface DAPMessage {
    seq: number;
    type: 'request' | 'response' | 'event';
    command?: string;
    event?: string;
    request_seq?: number;
    success?: boolean;
    body?: any;
    arguments?: any;
}

export class GojaDebugSession extends LoggingDebugSession {
    private static threadID = 1;
    private _variableHandles = new Handles<string>();
    private _configurationDone = new Subject();
    private _debugServerPort: number = 0;
    private _debugServerClient?: Net.Socket;
    private _gojaProcess?: ChildProcess;
    private _messageBuffer: string = '';
    private _seq: number = 1;
    private _dapPendingRequests = new Map<number, (response: any) => void>();
    private _isAttach: boolean = false;
    private _program?: string;

    public constructor() {
        super("goja-debug.txt");

        this.setDebuggerLinesStartAt1(true);
        this.setDebuggerColumnsStartAt1(true);
    }

    protected initializeRequest(response: DebugProtocol.InitializeResponse, args: DebugProtocol.InitializeRequestArguments): void {
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
        this.sendEvent(new InitializedEvent());
    }

    protected configurationDoneRequest(response: DebugProtocol.ConfigurationDoneResponse, args: DebugProtocol.ConfigurationDoneArguments): void {
        super.configurationDoneRequest(response, args);
        
        // Send configurationDone to DAP server
        this.sendDAPRequest('configurationDone', {}, (dapResponse) => {
            this._configurationDone.notify();
        });
        
        this.sendResponse(response);
    }

    protected async launchRequest(response: DebugProtocol.LaunchResponse, args: any) {
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

    protected async attachRequest(response: DebugProtocol.AttachResponse, args: any) {
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

    private async connectToDebugServer(): Promise<void> {
        return new Promise((resolve, reject) => {
            this._debugServerClient = Net.connect(this._debugServerPort, 'localhost');

            this._debugServerClient.on('connect', () => {
                resolve();
            });

            this._debugServerClient.on('error', (err) => {
                reject(err);
            });

            this._debugServerClient.on('close', () => {
                this.sendEvent(new TerminatedEvent());
            });

            this._debugServerClient.on('data', (data) => {
                this.handleDAPData(data);
            });
        });
    }

    private handleDAPData(data: Buffer) {
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
                const dapMessage: DAPMessage = JSON.parse(message);
                this.handleDAPMessage(dapMessage);
            } catch (e) {
                console.error('Failed to parse DAP message:', e);
            }
        }
    }

    private handleDAPMessage(message: DAPMessage) {
        if (message.type === 'response') {
            const handler = this._dapPendingRequests.get(message.request_seq!);
            if (handler) {
                this._dapPendingRequests.delete(message.request_seq!);
                handler(message);
            }
        } else if (message.type === 'event') {
            switch (message.event) {
                case 'stopped':
                    const stoppedBody = message.body;
                    this.sendEvent(new StoppedEvent(
                        stoppedBody.reason || 'breakpoint',
                        stoppedBody.threadId || GojaDebugSession.threadID
                    ));
                    break;
                case 'output':
                    const outputBody = message.body;
                    this.sendEvent(new OutputEvent(
                        outputBody.output,
                        outputBody.category
                    ));
                    break;
                case 'terminated':
                    this.sendEvent(new TerminatedEvent());
                    break;
            }
        }
    }

    private sendDAPRequest(command: string, args: any, handler?: (response: DAPMessage) => void): void {
        const request: DAPMessage = {
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

    private sendDAPRequestAsync(command: string, args: any): Promise<DAPMessage> {
        return new Promise((resolve) => {
            this.sendDAPRequest(command, args, resolve);
        });
    }

    protected async setBreakPointsRequest(response: DebugProtocol.SetBreakpointsResponse, args: DebugProtocol.SetBreakpointsArguments) {
        const path = args.source.path!;
        const clientBreakpoints = args.breakpoints || [];
        const clientLines = args.lines || [];

        // Send to DAP server
        const dapResponse = await this.sendDAPRequestAsync('setBreakpoints', {
            source: { path: path },
            breakpoints: clientBreakpoints.length > 0 
                ? clientBreakpoints 
                : clientLines.map(line => ({ line }))
        });

        const breakpoints: DebugProtocol.Breakpoint[] = [];
        if (dapResponse.body?.breakpoints) {
            dapResponse.body.breakpoints.forEach((bp: any) => {
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

    protected threadsRequest(response: DebugProtocol.ThreadsResponse): void {
        this.sendDAPRequest('threads', {}, (dapResponse) => {
            if (dapResponse.body?.threads) {
                response.body = {
                    threads: dapResponse.body.threads
                };
            } else {
                response.body = {
                    threads: [new Thread(GojaDebugSession.threadID, "main")]
                };
            }
            this.sendResponse(response);
        });
    }

    protected stackTraceRequest(response: DebugProtocol.StackTraceResponse, args: DebugProtocol.StackTraceArguments): void {
        this.sendDAPRequest('stackTrace', {
            threadId: args.threadId,
            startFrame: args.startFrame,
            levels: args.levels
        }, (dapResponse) => {
            if (dapResponse.body) {
                response.body = dapResponse.body;
            } else {
                response.body = {
                    stackFrames: [],
                    totalFrames: 0
                };
            }
            this.sendResponse(response);
        });
    }

    protected scopesRequest(response: DebugProtocol.ScopesResponse, args: DebugProtocol.ScopesArguments): void {
        this.sendDAPRequest('scopes', {
            frameId: args.frameId
        }, (dapResponse) => {
            if (dapResponse.body?.scopes) {
                response.body = {
                    scopes: dapResponse.body.scopes
                };
            } else {
                response.body = {
                    scopes: []
                };
            }
            this.sendResponse(response);
        });
    }

    protected variablesRequest(response: DebugProtocol.VariablesResponse, args: DebugProtocol.VariablesArguments): void {
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
            } else {
                response.body = {
                    variables: []
                };
            }
            this.sendResponse(response);
        });
    }

    protected continueRequest(response: DebugProtocol.ContinueResponse, args: DebugProtocol.ContinueArguments): void {
        this.sendDAPRequest('continue', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }

    protected nextRequest(response: DebugProtocol.NextResponse, args: DebugProtocol.NextArguments): void {
        this.sendDAPRequest('next', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }

    protected stepInRequest(response: DebugProtocol.StepInResponse, args: DebugProtocol.StepInArguments): void {
        this.sendDAPRequest('stepIn', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }

    protected stepOutRequest(response: DebugProtocol.StepOutResponse, args: DebugProtocol.StepOutArguments): void {
        this.sendDAPRequest('stepOut', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }

    protected evaluateRequest(response: DebugProtocol.EvaluateResponse, args: DebugProtocol.EvaluateArguments): void {
        this.sendDAPRequest('evaluate', {
            expression: args.expression,
            frameId: args.frameId,
            context: args.context
        }, (dapResponse) => {
            if (dapResponse.body) {
                response.body = dapResponse.body;
            } else {
                response.body = {
                    result: 'Error',
                    variablesReference: 0
                };
            }
            this.sendResponse(response);
        });
    }

    protected pauseRequest(response: DebugProtocol.PauseResponse, args: DebugProtocol.PauseArguments): void {
        this.sendDAPRequest('pause', {
            threadId: args.threadId
        }, () => {
            this.sendResponse(response);
        });
    }

    protected disconnectRequest(response: DebugProtocol.DisconnectResponse, args: DebugProtocol.DisconnectArguments): void {
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

class Subject {
    private _callbacks: (() => void)[] = [];

    notify(): void {
        this._callbacks.forEach(cb => cb());
        this._callbacks = [];
    }

    wait(timeout: number): Promise<void> {
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