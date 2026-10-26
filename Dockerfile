FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY web/ ./
RUN yarn build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o casnode .

FROM node:22-alpine
WORKDIR /app
COPY --from=backend /src/casnode ./
COPY --from=backend /src/conf ./conf
COPY --from=web /web/build ./web/build
ENV dbName=/app/data/casnode.db uploadsDir=/app/data/uploads
VOLUME /app/data
EXPOSE 16900
ENTRYPOINT ["./casnode"]
