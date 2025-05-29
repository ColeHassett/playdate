# PlayDate

Schedule dates to play games with your friends. 🙂

## Dependencies
* go
* docker or podman
* [just](https://github.com/casey/just?tab=readme-ov-file#installation)

## Running Locally

Copy the `.env` file and replace the environment variables with new defaults or just keep the defaults

```shell
cp .env.example .env
```

To run locally start the postgres instance:
```shell
docker-compose up postgres
```

This should start a local instance of postgres available at `localhost:5432`. Note it is configured with the environment variables from your local `.env`.

Finally start the app using just
```shell
just
```

