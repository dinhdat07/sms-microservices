# SMS Microservices C4 Models

This directory contains the Structurizr DSL (`workspace.dsl`) for the Server Management System (SMS) Microservices architecture.

## Usage

You can use the official [Structurizr Lite](https://structurizr.com/help/lite) tool to view and edit these diagrams locally.

### Running Structurizr Lite via Docker

Run the following command from this `c4` directory:

```bash
docker run -it --rm -p 8080:8080 -v $PWD:/usr/local/structurizr structurizr/lite
```

Then open `http://localhost:8080` in your web browser.

### Contents

- `workspace.dsl`: The authoritative C4 model encompassing System Context, Containers, and internal Microservice Components (Identity, Management, Reporting).
