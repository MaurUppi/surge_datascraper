# Surge Recent data Dashboard

Get recent data by Surge HTTP-API and set up the Grafana dashboard. 


### How to use

1. modify `docker-compose.yaml` 
2. start `docker-compose run influxdb`   #Optional if you changed my default settings. 
3. Open the browser to http://influxdb-ip-address:8086 to set up TOKEN.   #Optional if you changed my default settings. 
4. revise `.env`, and change settings accordingly, especially the `**MUST**` value. 
5. start `docker-compose up -d`
6. setup Grafana (add datasource)
7. import `panels/dashboard.json` and `panels/logs.json` to Grafana


### Dashboard Screenshot

![screenshot](./screenshot/screenshot.jpg)


#### InfluxDB TOKEN setting example

![screenshot](./screenshot/TOKEN.jpeg)


#### Grafana data source setting example

![screenshot](./screenshot/datasource.jpeg)


