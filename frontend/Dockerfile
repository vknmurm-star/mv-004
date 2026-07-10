# syntax=docker/dockerfile:1.6
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
RUN printf 'server {\n  listen 80;\n  server_name _;\n  root /usr/share/nginx/html;\n  location / { try_files $uri /index.html; }\n  location /api/ { proxy_pass http://backend:8080; proxy_set_header Host $host; }\n}\n' > /etc/nginx/conf.d/default.conf
EXPOSE 80