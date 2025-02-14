# Receipt Processor API

This project implements a simple REST API for processing receipts and calculating points based on a set of rules.

## Project Structure

*   **`main.go`:**  Contains the Go source code for the API.  This includes the HTTP handlers, point calculation logic, and an in-memory data store.
*   **`main_test.go`:** Contains unit tests for the API, covering various scenarios and edge cases.  This demonstrates a commitment to code quality and correctness.
*   **`Dockerfile`:**  Defines how to build a Docker image for the application, ensuring a consistent and reproducible environment.
*   **`api.yml`:** (Provided in the original challenge) Describes the API endpoints in a formal way.

## API Endpoints

The API has two endpoints, as described in the original challenge:

*   **`POST /receipts/process`:**  Processes a receipt (submitted as JSON) and returns a unique ID for that receipt.
*   **`GET /receipts/{id}/points`:** Retrieves the number of points awarded for a receipt, given its ID.

## Point Calculation Rules

The points are calculated based on these rules (taken directly from the challenge description):

*   One point for every alphanumeric character in the retailer name.
*   50 points if the total is a round dollar amount with no cents.
*   25 points if the total is a multiple of `0.25`.
*   5 points for every two items on the receipt.
*   If the trimmed length of the item description is a multiple of 3, multiply the price by `0.2` and round up to the nearest integer. The result is the number of points earned.
*   6 points if the day in the purchase date is odd.
*   10 points if the time of purchase is after 2:00 PM and before 4:00 PM.

## Running the Application (Two Methods)

This project provides two ways to run the application: using Docker (recommended for production and consistent environments) and directly with Go (for development and quick testing).

### Method 1: Using Docker (Recommended)

This is the easiest and most reliable way to run the application, as it eliminates any potential dependency or environment issues.  You need to have [Docker Desktop](https://www.docker.com/products/docker-desktop) (or a compatible Docker engine) installed.

1.  **Build the Docker Image:**

    Open a terminal (PowerShell on Windows, or a regular terminal on macOS/Linux) in the directory containing the `Dockerfile` and `main.go` files.  Run the following command:

    ```bash
    docker build -t receipt-processor .
    ```

    This command builds a Docker image named `receipt-processor` based on the instructions in the `Dockerfile`.  The `.` tells Docker to look for the `Dockerfile` in the current directory.

2.  **Run the Docker Container:**

    ```bash
    docker run -p 8080:8080 receipt-processor
    ```

    This command runs the Docker image in a container and maps port 8080 on your host machine to port 8080 inside the container.  This means you'll access the API at `http://localhost:8080`. The `-p 8080:8080` part is crucial for port mapping.

3.  **Stop the Container (When Finished):**

     To stop the running container, you first need to find its ID. Open *another* terminal and run:

    ```bash
    docker ps
    ```
    This will list all running containers.  Look for the container with the IMAGE name `receipt-processor`.  Then, stop it using:

    ```bash
    docker stop <container_id>
    ```

    Replace `<container_id>` with the actual container ID (a long string of hexadecimal characters).  Alternatively, you can stop all running containers with `docker stop $(docker ps -q)`, but be careful with this if you have other containers you want to keep running.

### Method 2: Running with Go Directly (For Development)

1.  **Install Go:** 

2.  **Install Dependencies:**
    Open a terminal in the directory containing `main.go`. Run these commands to download the required Go packages:

    ```bash
    go get [github.com/google/uuid](https://github.com/google/uuid)
    go get [github.com/gorilla/mux](https://github.com/gorilla/mux)
    ```

3.  **Run the Application:**

    ```bash
    go run main.go
    ```

    This command compiles and runs your `main.go` file.  The server will start listening on port 8080.

4. **Stop the Server (When Finished):**
   Press `Ctrl+C` in the terminal where you ran `go run main.go`.

## Testing the Application

Once the server is running (using either Docker or `go run`), you can interact with it using `curl` (or any other HTTP client, like Postman).

**1. Process a Receipt (POST Request):**

The best way to send JSON data with `curl` is to use a "here-string":

```bash
curl -X POST -H "Content-Type: application/json" -d @- http://localhost:8080/receipts/process <<EOF
{
  "retailer": "Target",
  "purchaseDate": "2022-01-01",
  "purchaseTime": "13:01",
  "items": [
    {
      "shortDescription": "Mountain Dew 12PK",
      "price": "6.49"
    },{
      "shortDescription": "Emils Cheese Pizza",
      "price": "12.25"
    },{
      "shortDescription": "Knorr Creamy Chicken",
      "price": "1.26"
    },{
      "shortDescription": "Doritos Nacho Cheese",
      "price": "3.35"
    },{
      "shortDescription": "Klarbrunn 12-PK 12 FL OZ",
      "price": "12.00"
    }
  ],
  "total": "35.35"
}
EOF

```
This command will output a JSON response containing the ID of the processed receipt:

JSON

{"id": "a-generated-uuid"}
2. Get Points for a Receipt (GET Request):

Replace <receipt_id> with the actual ID you received from the POST request:

Bash

curl http://localhost:8080/receipts/<receipt_id>/points
This will return a JSON response containing the number of points awarded:

{"points": 28}
3. Running Unit Tests:
The project includes comprehensive unit tests. To run the tests:

With Go: go test -v
With Docker: docker run -it receipt-processor go test -v

Key Design Decisions and Optimizations

Integer Cents: The code stores prices and totals in cents as integers, rather than using floating-point numbers. This avoids potential floating-point precision issues and is something that I find to be generally more efficient for monetary calculations.
strings.TrimSpace: The code uses strings.TrimSpace to remove leading and trailing whitespace from all input strings. This makes the API more robust to variations in input formatting.
Pre-parsed Times: The init() function pre-parses the "2:00 PM" and "4:00 PM" time strings to avoid re-parsing them on every request.
Error Handling: The code includes error handling, returning appropriate HTTP status codes (400 Bad Request, 404 Not Found, 500 Internal Server Error) and logging error messages. This is included because of its criticality in reliable APIs and I felt it would be relevant to demonstrate that understanding.
uuid for IDs: The code uses the github.com/google/uuid package to generate unique IDs for receipts.
Direct JSON encoding uses the encoder directly instead of making temporary variables.

Parallelization was considered, but honestly I figured it'd be a little overkill for reciept handling and may have possibly slowed down small scale calculations by introducing additional computational overhead.


Dependencies
The project uses the following external Go packages:

github.com/google/uuid: For generating unique receipt IDs.
github.com/gorilla/mux: For routing HTTP requests. This is a popular and well-maintained routing library for Go.
These dependencies are managed using Go modules and are automatically downloaded when you build the project (either with go build or docker build).