# Example Node.js Postgres
This example app shows how to run Postgres locally and connect to it using Node.js. We provide this example app
to avoid the hassle of researching these aspects, as we don't believe that this is time well spent on the assignment.

## Starting a Postgres Instance
This example app uses a Postgres database. You can start a Postgres instance using Docker by running the following command:

```bash
pnpm run postgres
```

## Initializing the Database
Within the `initdb/` directory you can find database initialization scripts. The scripts themselves are just examples
for now. Whenever Postgres is started for the first time (not restarted!), the `initdb/` scripts are executed. We
expect you to modify these scripts to suit the assignment.

## Connecting to the Database
Within the `index.js` file we provide an example showing how to connect and read information from the database. First
things first though, you need to install the dependencies:

```bash
pnpm install
```

Then you need to configure the database connection. You can do this by setting the following environment variables. The
example supports [dotenv](https://www.npmjs.com/package/dotenv).

```
PGUSER=postgres
PGPASSWORD=examplepw
PGHOST=127.0.0.1
PGDATABASE=postgres
PGPORT=5432
```

Then you can run the example:

```bash
pnpm start
```

This will connect to the database and read the contents of the `example` table. It should print something like this:

```yaml
[ { id: 1, msg: 'Hello, world!' } ]
```