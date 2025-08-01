import * as vscode from 'vscode';
import * as Net from 'net';
import { GojaDebugSession } from './gojaDebug';

export function activate(context: vscode.ExtensionContext) {
    console.log('Goja debugger extension activated');

    // Register a configuration provider for 'goja' debug type
    const provider = new GojaConfigurationProvider();
    context.subscriptions.push(vscode.debug.registerDebugConfigurationProvider('goja', provider));

    // Register the debug adapter factory
    const factory = new InlineDebugAdapterFactory();
    context.subscriptions.push(vscode.debug.registerDebugAdapterDescriptorFactory('goja', factory));
}

export function deactivate() {
    // Nothing to do
}

class GojaConfigurationProvider implements vscode.DebugConfigurationProvider {
    resolveDebugConfiguration(folder: vscode.WorkspaceFolder | undefined, config: vscode.DebugConfiguration): vscode.ProviderResult<vscode.DebugConfiguration> {
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

class InlineDebugAdapterFactory implements vscode.DebugAdapterDescriptorFactory {
    createDebugAdapterDescriptor(session: vscode.DebugSession): vscode.ProviderResult<vscode.DebugAdapterDescriptor> {
        // When using a debug server, connect to it
        if (session.configuration.debugServer) {
            return new vscode.DebugAdapterServer(session.configuration.debugServer);
        }

        // Otherwise use inline adapter
        return new vscode.DebugAdapterInlineImplementation(new GojaDebugSession());
    }
}