# PlayDate

Schedule dates to play games with your friends. 🙂

# Contributing

## Dependencies
* go
* docker or podman
* equivalent compose runner
* _optional_: [just](https://github.com/casey/just?tab=readme-ov-file#installation)

## Running Locally

Copy the `.env` file and replace the environment variables with new defaults or just keep the defaults

```shell
cp .env.example .env
```

To run locally start using the provider docker compose
```shell
docker-compose up --build
```

The http server will be available at `localhost:8080`.

And thats it! Now you can change files locally and your go http server will live reload based on them without having to restart your docker compose command or rebuilding your entire docker image.

### Using Air on Windows with Docker

You will need to set ```poll = true``` in your .air.toml file on Windows for live reload to work.

## Optional: Running PlayDate Using Just

**Note** if you are lazy you can use just instead of all of that typing
```shell
just
```

### Using Powershell on Windows with Just

You will need to add the following line to the top of your justfile if using powershell to run just:
```shell
set shell := ["powershell.exe", "-c"]
```


