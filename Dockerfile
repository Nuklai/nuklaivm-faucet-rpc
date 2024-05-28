# Use the official Golang image as the base image
FROM golang:1.21

# Set the working directory
WORKDIR /app

# Copy the Go application
COPY . .

# Build the Go application
RUN go build -o faucet

# Create .env file

RUN chmod +x ./infra/scripts/startup.sh
ENTRYPOINT [ "./infra/scripts/startup.sh" ]

# Expose the application port
EXPOSE 10591

# Command to run the application
CMD ["./faucet"]
