# TONX take-home assignment

## Development Guide

### Prerequisites

- Go (1.23)
- Make
- Docker

## Setup Instructions

1. Build and Run the Application:

  - Ensure Docker is installed and running.
  - Build and start the containers:

    ```bash
    make up
    ```

  - To stop and remove the containers:

    ```bash
    make down
    ```

2. Access the App:

  - Backend API: `http://localhost:8080`

## API Documentation

The API specification is defined using OpenAPI 3.0 and can be found in:

```
api/api.yaml
```

## Notes

The current implementation focuses on demonstrating the core order submission process with proper concurrency control. For clarity and brevity, several aspects have been simplified:

### Simplified Validations

- Flight status validation (e.g., checking if flight is open for booking)
- Customer information and status verification

### Omitted Features

- Seat selection and assignment
- Information of additional travelers in the same booking

### Future Enhancements

These features can be added by introducing additional tables:

- `OrderSeats`: Map selected seats to orders
- `OrderTravelers`: Store information of all travelers in an order

## Flight Booking Order Creation Flow

### Check and Reserve Seats

1. Try to get available seats from Redis

  - If Redis key doesn't exist:

    - Get flight from DB
    - Initialize Redis with flight's available seats using SetNX

  - If seats = 0, return no seats error

2. Use Lua script to check and decrement seats in Redis

  - Script returns -1: key not found
  - Script returns 0: no available seats
  - Script returns 1: success

3. Set up Redis restoration in case of later failure

  - Will restore original seats value if anything fails
  - Won't restore if transaction succeeds

### Create Order (Database Transaction)

1. Start transaction

2. Lock and get flight record using SELECT FOR UPDATE

3. Double check if enough seats available

  - Return error if not enough seats

4. Create order record

5. Update flight's available seats

6. Commit transaction

7. Mark Redis restoration as not needed (success case)

### Error Handling

- Any error before Redis decrement: return error
- Any error after Redis decrement: restore Redis seats, return error
- Any error in transaction: rollback transaction, restore Redis seats

Core business logic at `internal/service/order.go`.

Tests for the logic at `internal/service/order_test.go`

## Getting Started

1. Install dependencies:

  ```bash
  make dependencies
  ```

2. Making API Changes

After making changes to `api/api.yaml`, you need to regenerate the API structures:

```bash
make generate
```

This will update the request and response body structures based on your OpenAPI definitions.

## Important Notes

- Always run `make generate` after modifying the API specification
- Commit both the API specification and generated code changes
