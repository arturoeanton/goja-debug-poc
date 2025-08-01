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
exports.deactivate = exports.activate = void 0;
const vscode = __importStar(require("vscode"));
const gojaDebug_1 = require("./gojaDebug");
function activate(context) {
    console.log('Goja debugger extension activated');
    // Register a configuration provider for 'goja' debug type
    const provider = new GojaConfigurationProvider();
    context.subscriptions.push(vscode.debug.registerDebugConfigurationProvider('goja', provider));
    // Register the debug adapter factory
    const factory = new InlineDebugAdapterFactory();
    context.subscriptions.push(vscode.debug.registerDebugAdapterDescriptorFactory('goja', factory));
}
exports.activate = activate;
function deactivate() {
    // Nothing to do
}
exports.deactivate = deactivate;
class GojaConfigurationProvider {
    resolveDebugConfiguration(folder, config) {
        // If launch.json is missing or empty
        if (!config.type && !config.request && !config.name) {
            const editor = vscode.window.activeTextEditor;
            if (editor && editor.document.languageId === 'javascript') {
                config.type = 'goja';
                config.name = 'Debug Goja Script';
                config.request = 'launch';
                config.program = '${file}';
                config.stopOnEntry = false;
                config.debugServer = 5678;
            }
        }
        if (!config.program) {
            return vscode.window.showInformationMessage("Cannot find a program to debug").then(_ => {
                return undefined;
            });
        }
        return config;
    }
}
class InlineDebugAdapterFactory {
    createDebugAdapterDescriptor(session) {
        // When using a debug server, connect to it
        if (session.configuration.debugServer) {
            return new vscode.DebugAdapterServer(session.configuration.debugServer);
        }
        // Otherwise use inline adapter
        return new vscode.DebugAdapterInlineImplementation(new gojaDebug_1.GojaDebugSession());
    }
}
//# sourceMappingURL=extension.js.map