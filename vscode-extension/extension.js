'use strict';

const childProcess = require('child_process');
const net = require('net');
const vscode = require('vscode');

const processes = new Set();

async function activate(context) {
  const factory = {
    async createDebugAdapterDescriptor(session) {
      const config = session.configuration;
      const port = config.port || await availablePort();
      const executable = config.c2c2Path || 'c2c2';
      const args = ['-n', '-q', `-dap-port=${port}`, config.program, ...(config.input || [])];
      const process = childProcess.spawn(executable, args, { stdio: 'inherit' });
      processes.add(process);
      process.once('exit', () => processes.delete(process));
      await waitForServer(process, port);
      return new vscode.DebugAdapterServer(port, '127.0.0.1');
    }
  };
  context.subscriptions.push(vscode.debug.registerDebugAdapterDescriptorFactory('casl2', factory));
  context.subscriptions.push({ dispose: () => processes.forEach(process => process.kill()) });
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

function waitForServer(process, port) {
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
        if (Date.now() < deadline && process.exitCode === null) {
          setTimeout(connect, 25);
        } else {
          reject(new Error(`c2c2 debug adapter did not start: ${error.message}`));
        }
      });
    };
    process.once('error', reject);
    process.once('exit', code => {
      if (Date.now() < deadline) {
        reject(new Error(`c2c2 exited before the debug session started (code ${code})`));
      }
    });
    connect();
  });
}

function deactivate() {
  processes.forEach(process => process.kill());
}

module.exports = { activate, deactivate };
