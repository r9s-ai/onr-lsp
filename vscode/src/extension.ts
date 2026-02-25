import * as vscode from "vscode";
import * as path from "path";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const cfg = vscode.workspace.getConfiguration("onrLsp");
  const configuredPath = cfg.get<string>("serverPath", "onr-lsp");

  const serverCommand = resolveServerPath(configuredPath, context);
  const serverOptions: ServerOptions = {
    command: serverCommand,
    args: [],
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "onr-dsl" }],
    synchronize: {
      configurationSection: "onrLsp",
    },
  };

  client = new LanguageClient("onr-lsp", "ONR LSP", serverOptions, clientOptions);
  context.subscriptions.push(client);
  await client.start();
}

export async function deactivate(): Promise<void> {
  if (!client) {
    return;
  }
  await client.stop();
}

function resolveServerPath(configuredPath: string, context: vscode.ExtensionContext): string {
  if (!configuredPath) {
    return "onr-lsp";
  }
  if (path.isAbsolute(configuredPath)) {
    return configuredPath;
  }
  if (configuredPath.includes(path.sep)) {
    return path.resolve(context.extensionPath, configuredPath);
  }
  return configuredPath;
}
