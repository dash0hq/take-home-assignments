import 'dotenv/config';
import pg from 'pg';

const client = new pg.Client()
await client.connect()

const res = await client.query('SELECT * FROM example')
console.log(res.rows)
await client.end()