import { spawn } from 'node:child_process';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const children = [];
let stopping = false;

const stop = (exitCode) => {
  if (stopping) return;
  stopping = true;
  process.exitCode = exitCode;
  for (const child of children) child.kill('SIGTERM');
};

for (const app of ['admin', 'client']) {
  const directory = path.join(root, 'frontend', app);
  const require = createRequire(path.join(directory, 'package.json'));
  const vite = path.join(
    path.dirname(require.resolve('vite/package.json')),
    'bin/vite.js',
  );
  const child = spawn(process.execPath, [vite, '--host'], {
    cwd: directory,
    stdio: 'inherit',
  });
  children.push(child);
  child.on('error', () => {
    console.error(`Не удалось запустить ${app}.`);
    stop(1);
  });
  child.on('exit', (code) => {
    if (!stopping) stop(code || 1);
  });
}

process.on('SIGINT', () => stop(0));
process.on('SIGTERM', () => stop(0));
