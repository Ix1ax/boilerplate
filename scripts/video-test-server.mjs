import fs from 'node:fs';
import https from 'node:https';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const args = process.argv.slice(2);
const option = (name, fallback) => {
  const at = args.indexOf(name);
  return at < 0 ? fallback : args[at + 1];
};
const certificate = option('--cert');
const privateKey = option('--key');
if (!certificate || !privateKey) throw new Error('Укажите --cert и --key: локальный HTTPS нужен для камеры и защищённых cookie');
const origin = new URL(option('--origin', 'https://localhost:9443'));
const gateway = new URL(option('--gateway', 'http://127.0.0.1:8080'));
const livekit = new URL(option('--livekit', 'http://127.0.0.1:7880'));
const appOrigin = new URL(option('--app-origin', 'http://localhost:3000')).origin;
const lan = args.includes('--lan');
if (origin.protocol !== 'https:' || origin.pathname !== '/' || origin.search || origin.hash || origin.username || origin.password) throw new Error('--origin: только https://хост:порт');
for (const upstream of [gateway, livekit]) {
  if (!['http:', 'https:'].includes(upstream.protocol) || upstream.username || upstream.password || upstream.search || upstream.hash || upstream.pathname !== '/') throw new Error('Upstream: только http(s)://хост:порт');
}
if (!lan && !['localhost', '127.0.0.1', '[::1]'].includes(origin.hostname)) throw new Error('Для адреса локальной сети явно добавьте --lan');
const page = fs.readFileSync(path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../tools/video-test.html'));
const allowed = (p, method) =>
  p === '/readyz' && method === 'GET' ||
  /^\/auth\/local\/(login|register)$/.test(p) && method === 'POST' ||
  p === '/api/v1/sports' && method === 'GET' ||
  p === '/api/v1/events' && method === 'POST' ||
  /^\/api\/v1\/events\/[0-9a-f-]{36}$/.test(p) && method === 'GET' ||
  /^\/api\/v1\/events\/[0-9a-f-]{36}\/format$/.test(p) && method === 'PUT' ||
  /^\/api\/v1\/events\/[0-9a-f-]{36}\/(join|video\/token)$/.test(p) && method === 'POST';
const send = (res, status, text) => {
  res.writeHead(status, {'content-type': 'text/plain; charset=utf-8', 'cache-control': 'no-store'});
  res.end(text);
};
const validHost = (req) => req.headers.host === origin.host;
const exactOrigin = (req) => req.headers.origin === origin.origin;
const headersFor = (req, target, media) => {
  const headers = {...req.headers, host: target.host};
  delete headers['forwarded'];
  delete headers['x-forwarded-for'];
  delete headers['x-forwarded-host'];
  delete headers['x-forwarded-proto'];
  if (media) {
    delete headers.cookie;
  } else {
    delete headers.authorization;
    delete headers['x-internal-token'];
    headers.origin = appOrigin;
  }
  return headers;
};
function proxy(req, res, target, pathname, media) {
  const transport = target.protocol === 'https:' ? https : http;
  const upstream = transport.request(new URL(pathname, target), {method:req.method, headers:headersFor(req,target,media)}, (response) => {
    const headers = {...response.headers, 'cache-control':'no-store'};
    delete headers['access-control-allow-origin'];
    // Cookie остаётся на helper origin. Domain backend никогда не переносим.
    if (headers['set-cookie']) headers['set-cookie'] = headers['set-cookie'].map((cookie) => cookie.replace(/;\s*Domain=[^;]+/ig,''));
    res.writeHead(response.statusCode || 502, headers);
    response.pipe(res);
  });
  upstream.setTimeout(30_000, () => upstream.destroy());
  upstream.on('error', () => { if (!res.headersSent) send(res,502,'Сервер недоступен'); else res.destroy(); });
  req.on('aborted', () => upstream.destroy());
  req.pipe(upstream);
}
const server = https.createServer({cert:fs.readFileSync(certificate), key:fs.readFileSync(privateKey)}, (req,res) => {
  if (!validHost(req)) return send(res,421,'Неверный адрес helper');
  const url = new URL(req.url,origin);
  if (url.pathname === '/' && req.method === 'GET') {
    res.writeHead(200, {'content-type':'text/html; charset=utf-8','cache-control':'no-store','x-content-type-options':'nosniff','referrer-policy':'no-referrer'});
    return res.end(page);
  }
  // Проверяем настоящий Origin до подстановки Origin приложения для gateway CSRF.
  if (!['GET','HEAD'].includes(req.method) && !exactOrigin(req)) return send(res,403,'Origin не совпадает с адресом helper');
  if (url.pathname === '/livekit/rtc' || url.pathname.startsWith('/livekit/rtc/')) {
    if (req.headers.origin && !exactOrigin(req)) return send(res,403,'Недопустимый Origin');
    return proxy(req,res,livekit,url.pathname.slice('/livekit'.length)+url.search,true);
  }
  if (!allowed(url.pathname,req.method) || url.search) return send(res,404,'Маршрут недоступен в тестовом helper');
  proxy(req,res,gateway,url.pathname,false);
});
server.on('upgrade',(req,socket,head) => {
  const url = new URL(req.url,origin);
  if (!validHost(req) || !exactOrigin(req) || url.pathname !== '/livekit/rtc') return socket.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n');
  const transport = livekit.protocol === 'https:' ? https : http;
  const upstream = transport.request(new URL('/rtc'+url.search,livekit),{method:'GET',headers:headersFor(req,livekit,true)});
  upstream.on('upgrade',(response,upstreamSocket,upstreamHead) => {
    socket.write(`HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\n`);
    for (const [name,value] of Object.entries(response.headers)) {
      for (const item of Array.isArray(value)?value:[value]) if (item !== undefined) socket.write(`${name}: ${item}\r\n`);
    }
    socket.write('\r\n');
    if (head.length) upstreamSocket.write(head);
    if (upstreamHead.length) socket.write(upstreamHead);
    socket.pipe(upstreamSocket).pipe(socket);
    socket.on('error',()=>upstreamSocket.destroy());
    upstreamSocket.on('error',()=>socket.destroy());
    socket.on('close',()=>upstreamSocket.destroy());
  });
  upstream.on('response',()=>socket.end('HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n'));
  upstream.on('error',()=>socket.destroy());
  socket.on('error',()=>upstream.destroy());
  upstream.end();
});
server.listen(Number(origin.port || 443),lan?'0.0.0.0':'127.0.0.1',()=>console.log(`Видеопроверка: ${origin.origin}${lan?' (доступна в локальной сети)':''}`));
