# Storm-Hunt
My project about storm-hunting (it's barely started, but I'll develop it!)

<<<<<<< HEAD
Now the application just streaming a weather data about some regions by using Open API. To start app, choose some directory on your computer, then open cmd and open path to this directory (set cd + yourpath/to/directory). After that make command:

git clone https://github.com/KarabasUehal/Storm-Hunt.git

Await for download app. Before starting, you should create .env file in root of project. Insert your data into variables:

OPENWEATHER_API_KEY=*yourapikey*
MYSQL_USER=*youruser*
MYSQL_PASSWORD=*yourpass*
MYSQL_DBNAME=*yourdbname*

MYSQL_HOST=mysql 
MYSQL_PORT=3306
MYSQL_ROOT_PASSWORD=rootpass
KEYCLOAK_URL=http://storm-keycloak:8080
GRPC_PORT=50051
REST_PORT=8082
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_ADDR=redis:6379
RABBITMQ_USER=myuser444
RABBITMQ_PASS=mypass444
RABBITMQ_URL=amqp://myuser444:mypass444@rabbitmq:5672/
REGIONS=Atlantic,Pacific

Make all notes without space beetwen words and signs. Now you need to get your API key - it's fast! Go to: 

https://openweathermap.org/

And register your account. After registration click on your profile at upper right corner and choose "My API keys". Copy it and set into your .env file (OPENWEATHER_API_KEY).

Then choose directory with this app (set cd + yourpath/to/appfolder) in cmd and set command:
=======
Now the application just streaming a weather data about some regions by using Open API. To start app, run Docker on your computer, then open cmd, choose directory with this app (set cd + yourpath/to/appfolder) and set command:
>>>>>>> d7ba0749d58cf4c5c23b33c22a82850b1907fac4

docker-compose up --build

Await for starting docker-compose. Soon you will get ready application!

Open browser and follow next adress:

localhost:3000

You will redirected to localhost:8081 to keycloak service. Register your account to continue. Then log in.

Now you will see the application that streaming weather-data about Miami (Atlantic ocean) and Honolulu (Pacific ocean). You can start and stop choosen stream by using button and logout if you want to return to authentification page. Soon I'll add more features!

Thanks for reading!
