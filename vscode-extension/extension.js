'use strict';

const net = require('net');
const vscode = require('vscode');

const terminals = new Set();

async function activate(context) {
  const factory = {
    async createDebugAdapterDescriptor(session) {
      const config = session.configuration;
      const port = config.port || await availablePort();
      const executable = config.c2c2Path || 'c2c2';
      const args = ['-n', '-q', `-dap-port=${port}`, config.program, ...(config.input || [])];
      const terminal = vscode.window.createTerminal({
        name: `CASL2: ${config.program}`,
        shellPath: executable,
        shellArgs: args
      });
      terminals.add(terminal);
      terminal.show();
      await waitForServer(port);
      return new vscode.DebugAdapterServer(port, '127.0.0.1');
    }
  };
  context.subscriptions.push(vscode.debug.registerDebugAdapterDescriptorFactory('casl2', factory));
  context.subscriptions.push(vscode.window.onDidCloseTerminal(terminal => terminals.delete(terminal)));
  context.subscriptions.push({ dispose: () => terminals.forEach(terminal => terminal.dispose()) });
}

function availablePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const port = server.address().port;
      server.close(error => error ? reject(error) : resolve(port));
    });
  });
}

function waitForServer(port) {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + 5000;
    const connect = () => {
      const socket = net.connect(port, '127.0.0.1');
      socket.once('connect', () => {
        socket.end();
        resolve();
      });
      socket.once('error', error => {
        socket.destroy();
        if (Date.now() < deadline) {
          setTimeout(connect, 25);
        } else {
          reject(new Error(`c2c2 debug adapter did not start: ${error.message}`));
        }
      });
    };
    connect();
  });
}

function deactivate() {
  terminals.forEach(terminal => terminal.dispose());
}

module.exports = { activate, deactivate };
