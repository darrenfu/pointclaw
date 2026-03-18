FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /scraper ./cmd/scraper

FROM mcr.microsoft.com/playwright:v1.49.0-noble

COPY --from=builder /scraper /scraper

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
CMD ["/scraper"]
