FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
COPY client/package.json client/package.json
COPY admin/package.json admin/package.json
COPY packages/ui/package.json packages/ui/package.json
COPY packages/contracts/package.json packages/contracts/package.json
RUN npm ci
COPY . .
RUN npm run build:admin
FROM nginxinc/nginx-unprivileged:alpine AS runner
COPY admin/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/admin/dist /usr/share/nginx/html/admin
EXPOSE 8080
