// Собирает общий packages/contracts/openapi.yaml из спецификаций сервисов (server/*/api/openapi.yaml).
// Источник правды — файлы сервисов; общий файл генерируется, руками его не правят.
// Запуск: make openapi (заодно перегенерирует TS-типы и копию для Swagger в gateway).
import { readFileSync, readdirSync, existsSync, writeFileSync } from 'node:fs';
import { isDeepStrictEqual } from 'node:util';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse, stringify } from 'yaml';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
// gateway первым: его описание открывает общую документацию.
const services = [
  'gateway',
  ...readdirSync(path.join(root, 'server'))
    .filter(
      (s) =>
        s !== 'gateway' &&
        existsSync(path.join(root, 'server', s, 'api', 'openapi.yaml')),
    )
    .sort(),
];
const problems = [];
const merged = { paths: {}, components: {}, tags: [] };
let base;
for (const svc of services) {
  const spec = parse(
    readFileSync(path.join(root, 'server', svc, 'api', 'openapi.yaml'), 'utf8'),
  );
  base ??= spec;
  for (const [p, item] of Object.entries(spec.paths ?? {})) {
    if (merged.paths[p])
      problems.push(
        `путь ${p} описан и в ${merged.paths[p]['x-service']}, и в ${svc}`,
      );
    merged.paths[p] = { ...item, 'x-service': svc };
  }
  for (const tag of spec.tags ?? [])
    merged.tags.push({
      ...tag,
      description: `${tag.description ?? ''} (сервис ${svc})`.trim(),
    });
  for (const [kind, items] of Object.entries(spec.components ?? {})) {
    merged.components[kind] ??= {};
    for (const [name, value] of Object.entries(items)) {
      const prev = merged.components[kind][name];
      // Одинаковые общие компоненты (Error, ответы) сливаются; разные под одним именем — ошибка.
      if (prev && !isDeepStrictEqual(prev.value, value))
        problems.push(
          `components.${kind}.${name} отличается в ${prev.svc} и ${svc} — переименуйте или выровняйте`,
        );
      merged.components[kind][name] ??= { value, svc };
    }
  }
}
if (problems.length) {
  console.error(
    'Не удалось собрать общий OpenAPI:\n- ' + problems.join('\n- '),
  );
  process.exit(1);
}
for (const p of Object.values(merged.paths)) delete p['x-service'];
const components = Object.fromEntries(
  Object.entries(merged.components).map(([kind, items]) => [
    kind,
    Object.fromEntries(Object.entries(items).map(([n, v]) => [n, v.value])),
  ]),
);
const doc = {
  openapi: base.openapi,
  info: {
    title: 'API проекта — все сервисы',
    version: base.info.version,
    description:
      base.info.description +
      '\n**Сервисы** (отдельные спецификации — в списке сверху):\n\n' +
      services
        .map((s) => `- \`${s}\` — server/${s}/api/openapi.yaml`)
        .join('\n') +
      '\n',
  },
  servers: base.servers,
  security: base.security,
  tags: merged.tags,
  paths: merged.paths,
  components,
};
const out = path.join(root, 'packages', 'contracts', 'openapi.yaml');
writeFileSync(
  out,
  '# СГЕНЕРИРОВАНО scripts/openapi.mjs из server/*/api/openapi.yaml — не редактируйте вручную.\n' +
    '# Правьте спецификацию нужного сервиса и запускайте make openapi.\n' +
    stringify(doc, { lineWidth: 110 }),
);
console.log(
  `Общий OpenAPI собран: ${Object.keys(doc.paths).length} путей из ${services.length} сервисов.`,
);
