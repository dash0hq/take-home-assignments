import { createApp } from './app';
import { config } from './config';

const app = createApp();

app.listen(config.port, () => {
    console.log(JSON.stringify({ level: 'info', message: `Server started`, port: config.port }));
});
