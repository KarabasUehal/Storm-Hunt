# Storm-Hunt
My project about storm-hunting (it's barely started, but I'll develop it!)

Now the application just streaming a weather data about some regions by using Open API. To start app, run Docker on your computer, then open cmd, choose directory with this app (set cd + yourpath/to/appfolder) and set command:

docker-compose up --build

Await for starting docker-compose. Soon you will get ready application!

Open browser and follow next adress:

localhost:3000

You will redirected to localhost:8081 to keycloak service. Register your account to continue. Then log in.

Now you will see the application that streaming weather-data about Miami (Atlantic ocean) and Honolulu (Pacific ocean). You can start and stop choosen stream by using button and logout if you want to return to authentification page. Soon I'll add more features!

Thanks for reading!
