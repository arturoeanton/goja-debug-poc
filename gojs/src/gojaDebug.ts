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

export class GojaDebugSession extends LoggingDebugSession {
    private static threadID = 1;
    private _variableHandles = new Handles<string>();
    private _configurationDone = new Subject();
    private _debugServerPort: number = 0;
    private _debugServerClient?: Net.Socket;
    private _gojaProcess?: ChildProcess;

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
        this._configurationDone.notify();
    }

    protected async launchRequest(response: DebugProtocol.LaunchResponse, args: any) {
        await this._configurationDone.wait(1000);

        const program = args.program;
        const stopOnEntry = args.stopOnEntry || false;
        this._debugServerPort = args.debugServer || 5678;

        // Start gojs with debug flag
        const gojsPath = path.join(__dirname, '../../dap/gojs');
        const gojsArgs = ['-d', '-port', this._debugServerPort.toString(), '-f', program];

        this._gojaProcess = spawn('go', ['run', gojsPath + '.go', ...gojsArgs], {
            cwd: path.dirname(program)
        });

        this._gojaProcess.stdout?.on('data', (data) => {
            this.sendEvent(new OutputEvent(data.toString(), 'stdout'));
        });

        this._gojaProcess.stderr?.on('data', (data) => {
            this.sendEvent(new OutputEvent(data.toString(), 'stderr'));
        });

        this._gojaProcess.on('exit', (code) => {
            this.sendEvent(new TerminatedEvent());
        });

        // Give the debug server time to start
        await new Promise(resolve => setTimeout(resolve, 1000));

        // Connect to debug server
        await this.connectToDebugServer();

        this.sendResponse(response);
    }

    protected async attachRequest(response: DebugProtocol.AttachResponse, args: any) {
        this._debugServerPort = args.debugServer || 5678;
        await this.connectToDebugServer();
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
        });
    }

    protected setBreakPointsRequest(response: DebugProtocol.SetBreakpointsResponse, args: DebugProtocol.SetBreakpointsArguments): void {
        const path = args.source.path!;
        const clientLines = args.lines || [];

        const breakpoints = clientLines.map(l => {
            const bp: DebugProtocol.Breakpoint = {
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

    protected threadsRequest(response: DebugProtocol.ThreadsResponse): void {
        response.body = {
            threads: [
                new Thread(GojaDebugSession.threadID, "main")
            ]
        };
        this.sendResponse(response);
    }

    protected stackTraceRequest(response: DebugProtocol.StackTraceResponse, args: DebugProtocol.StackTraceArguments): void {
        const startFrame = typeof args.startFrame === 'number' ? args.startFrame : 0;
        const maxLevels = typeof args.levels === 'number' ? args.levels : 1000;

        const frames: StackFrame[] = [];
        // This would be populated from the debug adapter
        
        response.body = {
            stackFrames: frames,
            totalFrames: frames.length
        };
        this.sendResponse(response);
    }

    protected scopesRequest(response: DebugProtocol.ScopesResponse, args: DebugProtocol.ScopesArguments): void {
        const scopes: Scope[] = [];
        scopes.push(new Scope("Local", this._variableHandles.create("local"), false));

        response.body = {
            scopes: scopes
        };
        this.sendResponse(response);
    }

    protected variablesRequest(response: DebugProtocol.VariablesResponse, args: DebugProtocol.VariablesArguments): void {
        const variables: DebugProtocol.Variable[] = [];
        // This would be populated from the debug adapter

        response.body = {
            variables: variables
        };
        this.sendResponse(response);
    }

    protected continueRequest(response: DebugProtocol.ContinueResponse, args: DebugProtocol.ContinueArguments): void {
        this.sendResponse(response);
    }

    protected nextRequest(response: DebugProtocol.NextResponse, args: DebugProtocol.NextArguments): void {
        this.sendResponse(response);
    }

    protected stepInRequest(response: DebugProtocol.StepInResponse, args: DebugProtocol.StepInArguments): void {
        this.sendResponse(response);
    }

    protected stepOutRequest(response: DebugProtocol.StepOutResponse, args: DebugProtocol.StepOutArguments): void {
        this.sendResponse(response);
    }

    protected evaluateRequest(response: DebugProtocol.EvaluateResponse, args: DebugProtocol.EvaluateArguments): void {
        response.body = {
            result: 'Not implemented',
            variablesReference: 0
        };
        this.sendResponse(response);
    }

    protected pauseRequest(response: DebugProtocol.PauseResponse, args: DebugProtocol.PauseArguments): void {
        this.sendResponse(response);
    }

    protected disconnectRequest(response: DebugProtocol.DisconnectResponse, args: DebugProtocol.DisconnectArguments): void {
        if (this._debugServerClient) {
            this._debugServerClient.destroy();
        }
        if (this._gojaProcess) {
            this._gojaProcess.kill();
        }
        this.sendResponse(response);
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