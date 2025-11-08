import * as vscode from 'vscode';
import { WorkspaceFolder, DebugConfiguration, ProviderResult, CancellationToken } from 'vscode';
import * as Net from 'net';
import { spawn, ChildProcess } from 'child_process';

export function activate(context: vscode.ExtensionContext) {
	// Register the configuration provider
	const provider = new CASL2ConfigurationProvider();
	context.subscriptions.push(vscode.debug.registerDebugConfigurationProvider('casl2', provider));

	// Register the debug adapter descriptor factory
	const factory = new CASL2DebugAdapterDescriptorFactory();
	context.subscriptions.push(vscode.debug.registerDebugAdapterDescriptorFactory('casl2', factory));
}

export function deactivate() {
	// Cleanup if needed
}

class CASL2ConfigurationProvider implements vscode.DebugConfigurationProvider {
	/**
	 * Massage a debug configuration just before a debug session is being launched,
	 * e.g. add all missing attributes to the debug configuration.
	 */
	resolveDebugConfiguration(
		folder: WorkspaceFolder | undefined,
		config: DebugConfiguration,
		token?: CancellationToken
	): ProviderResult<DebugConfiguration> {
		// If launch.json is missing or empty
		if (!config.type && !config.request && !config.name) {
			const editor = vscode.window.activeTextEditor;
			if (editor && editor.document.languageId === 'casl2') {
				config.type = 'casl2';
				config.name = 'Debug CASL2 Program';
				config.request = 'launch';
				config.program = '${file}';
				config.stopOnEntry = true;
			}
		}

		if (!config.program) {
			return vscode.window.showInformationMessage('Cannot find a program to debug').then(_ => {
				return undefined; // abort launch
			});
		}

		// Set default values
		config.debugServer = config.debugServer || 4711;
		config.c2c2Path = config.c2c2Path || 'c2c2';
		config.stopOnEntry = config.stopOnEntry !== undefined ? config.stopOnEntry : true;

		return config;
	}
}

class CASL2DebugAdapterDescriptorFactory implements vscode.DebugAdapterDescriptorFactory {
	private serverProcess: ChildProcess | undefined;

	createDebugAdapterDescriptor(
		session: vscode.DebugSession,
		executable: vscode.DebugAdapterExecutable | undefined
	): ProviderResult<vscode.DebugAdapterDescriptor> {
		const config = session.configuration;
		const port = config.debugServer || 4711;
		const c2c2Path = config.c2c2Path || 'c2c2';

		// Start the c2c2 DAP server
		return this.startDAPServer(c2c2Path, port).then(() => {
			// Connect to the server
			return new vscode.DebugAdapterServer(port);
		}).catch(err => {
			vscode.window.showErrorMessage(`Failed to start CASL2 debug server: ${err.message}`);
			return undefined;
		});
	}

	private startDAPServer(c2c2Path: string, port: number): Promise<void> {
		return new Promise((resolve, reject) => {
			// Kill any existing server process
			if (this.serverProcess) {
				this.serverProcess.kill();
				this.serverProcess = undefined;
			}

			// Start the server
			this.serverProcess = spawn(c2c2Path, ['-dap', port.toString()]);

			this.serverProcess.on('error', (err) => {
				reject(new Error(`Failed to start c2c2: ${err.message}`));
			});

			// Wait for server to be ready
			const checkConnection = () => {
				const client = new Net.Socket();
				client.connect(port, '127.0.0.1', () => {
					client.end();
					resolve();
				});
				client.on('error', () => {
					// Retry after a short delay
					setTimeout(checkConnection, 100);
				});
			};

			// Start checking after a short delay
			setTimeout(checkConnection, 200);

			// Timeout after 5 seconds
			setTimeout(() => {
				reject(new Error('Timeout waiting for DAP server to start'));
			}, 5000);
		});
	}

	dispose() {
		if (this.serverProcess) {
			this.serverProcess.kill();
			this.serverProcess = undefined;
		}
	}
}
