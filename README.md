# ODB II Data Viewer

This is a project to view data provided by [Android OBD-II Reader](https://github.com/pires/android-obd-reader).

This is an example that stores all data it receives in memory. All data in memory is
shown in the view. No data is persisted anywhere.

```sh
$ obd2-data-viewer -port 9300
```

Set Android application to log to http://<your-server-ip>:9300/data

View data at http://<your-server-ip>:9300/
