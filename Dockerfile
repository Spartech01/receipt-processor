# Start from the official Go image.
FROM golang:1.21

# Set the working directory inside the container.
WORKDIR /app

# Copy the Go source code into the container.
COPY . .

# Install dependencies.
RUN go get github.com/google/uuid
RUN go get github.com/gorilla/mux

# Build the Go application.
RUN go build -o main .

# Expose port 8080 to the outside world.
EXPOSE 8080

# Command to run the executable.
CMD ["./main"]