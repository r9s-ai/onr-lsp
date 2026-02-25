import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const cfg = vscode.workspace.getConfiguration("onrLsp");
  const configuredPath = cfg.get<string>("serverPath", "");

  const serverCommand = resolveServerPath(configuredPath, context);
  const serverOptions: ServerOptions = {
    command: serverCommand,
    args: [],
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "onr-dsl", pattern: "**/providers/*.conf" },
    ],
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
  const trimmed = (configuredPath || "").trim();
  if (trimmed) {
    if (path.isAbsolute(trimmed)) {
      return trimmed;
    }
    if (trimmed.includes(path.sep)) {
      return path.resolve(context.extensionPath, trimmed);
    }
    return trimmed;
  }

  const bundled = bundledBinaryPath(context.extensionPath);
  if (bundled) {
    return bundled;
  }

  return "onr-lsp";
}

function bundledBinaryPath(extensionPath: string): string | undefined {
  const target = runtimeTarget();
  if (!target) {
    return undefined;
  }
  const exeName = target.platform === "win32" ? "onr-lsp.exe" : "onr-lsp";
  const p = path.join(extensionPath, "bin", `${target.platform}-${target.arch}`, exeName);
  if (!fs.existsSync(p)) {
    return undefined;
  }
  if (target.platform !== "win32") {
    try {
      fs.chmodSync(p, 0o755);
    } catch {
      // best effort
    }
  }
  return p;
}

function runtimeTarget(): { platform: string; arch: string } | undefined {
  const platform = process.platform;
  const arch = process.arch;
  if ((platform === "linux" || platform === "darwin" || platform === "win32") && (arch === "x64" || arch === "arm64")) {
    return { platform, arch };
  }
  return undefined;
}
