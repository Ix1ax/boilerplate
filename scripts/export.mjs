import { existsSync } from 'node:fs';
import { cp, mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const source = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const destination = process.argv[2] && path.resolve(process.argv[2]);
const name = process.argv[3] ?? 'hackathon-app';
if (!destination || !/^[a-z][a-z0-9-]{1,62}$/.test(name))
  throw new Error(
    'Использование: node scripts/export.mjs /новый/каталог имя-проекта. Имя: 2–63 символа, латиница, цифры и дефис.',
  );
if (
  destination === source ||
  destination.startsWith(source + path.sep) ||
  existsSync(destination)
)
  throw new Error(
    'Выберите новый каталог за пределами шаблона. Существующий каталог не перезаписывается.',
  );
const excluded = new Set([
  'node_modules',
  '.git',
  '.next',
  '.next-dev',
  '.next-preview',
  'dist',
  '.tmp',
  '.local',
  'bin',
  'coverage',
  '.DS_Store',
]);
await mkdir(path.dirname(destination), { recursive: true });
await cp(source, destination, {
  recursive: true,
  filter: (file) => {
    const relative = path.relative(source, file);
    const base = path.basename(file);
    return (
      !relative.split(path.sep).some((part) => excluded.has(part)) &&
      !(base.startsWith('.env') && base !== '.env.example') &&
      !base.endsWith('.tsbuildinfo') &&
      !/\.(pem|key)$/.test(base)
    );
  },
});
for (const file of ['package.json', 'package-lock.json']) {
  const target = path.join(destination, file);
  const json = JSON.parse(await readFile(target, 'utf8'));
  json.name = name;
  if (json.packages?.['']) json.packages[''].name = name;
  await writeFile(target, JSON.stringify(json, null, 2) + '\n');
}
for (const file of [
  '.env.example',
  'web/src/pages/home/ui/HomePage.tsx',
  'web/index.html',
  'infra/keycloak/realm.template.json',
]) {
  const target = path.join(destination, file);
  const text = await readFile(target, 'utf8');
  await writeFile(
    target,
    text
      .replaceAll('Hackathon starter', name)
      .replaceAll('hackathon-starter', name),
  );
}
console.log(
  `Шаблон создан: ${destination}. Для запуска выполните make up из нового каталога.`,
);
