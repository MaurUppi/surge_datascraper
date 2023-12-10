# Surge Recent data Dashboard

An example of a Surge HTTP-API.

### Screenshot

![screenshot](./screenshot/screenshot.jpeg)

### How to use

1. modify `docker-compose.yaml` 
2. start `docker-compose run influxdb`
3. Open browser to http://influxdb-ip-address:8086 to set up TOKEN.
4. revise `.env` , change settings accordingly, especially `**MUST**` value. 
4. start `docker-compose up -d`
5. setup Grafana (add datasource)
6. import `panels/dashboard.json` and `panels/logs.json` to Grafana


#### datasource setting example

![screenshot](./screenshot/datasource.jpeg)
