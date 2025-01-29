# use official Golang image
FROM golang:1.23.1-alpine

# RUN apk add - no-cache git make musl-dev go
# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest

# set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . . 

# Install dependencies
RUN go mod tidy

# Expose the application port
EXPOSE 8000

# Start the application with Air
CMD ["air"]
