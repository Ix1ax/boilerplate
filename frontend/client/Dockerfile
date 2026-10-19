FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
COPY frontend/client/package.json frontend/client/package.json
COPY frontend/admin/package.json frontend/admin/package.json
COPY packages/ui/package.json packages/ui/package.json
COPY packages/contracts/package.json packages/contracts/package.json
RUN npm ci
COPY . .
RUN npm run build:client
FROM nginxinc/nginx-unprivileged:alpine AS runner
COPY frontend/client/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/frontend/client/dist /usr/share/nginx/html
EXPOSE 8080
