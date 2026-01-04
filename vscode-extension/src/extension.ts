import * as path from 'path';
import * as fs from 'fs';
import * as vscode from 'vscode';
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
    console.log('CASL2 Language Support extension is now active');

    const config = vscode.workspace.getConfiguration('casl2');
    const enabled = config.get<boolean>('languageServer.enabled', true);
    
    if (!enabled) {
        console.log('CASL2 Language Server is disabled');
        return;
    }

    // Get the language server executable path from configuration
    let serverCommand = config.get<string>('languageServer.path', 'casl2-lsp');
    
    // If the path is relative or just a command name, try to resolve it
    if (!path.isAbsolute(serverCommand)) {
        // First, try to find it in the extension's directory
        const extensionServerPath = context.asAbsolutePath(path.join('bin', serverCommand));
        if (fs.existsSync(extensionServerPath)) {
            serverCommand = extensionServerPath;
        }
        // Otherwise, assume it's in the system PATH
    }

    console.log('CASL2 Language Server path:', serverCommand);

    // Server options for the language server
    const serverOptions: ServerOptions = {
        run: { command: serverCommand, transport: TransportKind.stdio },
        debug: { command: serverCommand, transport: TransportKind.stdio }
    };

    // Options to control the language client
    const clientOptions: LanguageClientOptions = {
        // Register the server for CASL2 documents
        documentSelector: [{ scheme: 'file', language: 'casl2' }],
        synchronize: {
            // Notify the server about file changes to CASL2 files in the workspace
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.{cas,casl,casl2}')
        }
    };

    // Create the language client and start it
    client = new LanguageClient(
        'casl2LanguageServer',
        'CASL2 Language Server',
        serverOptions,
        clientOptions
    );

    // Start the client (this will also launch the server)
    client.start();

    console.log('CASL2 Language Server started');
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
