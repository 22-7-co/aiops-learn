FROM debian:latest

WORKDIR /app
COPY . .

# install python3
RUN apt-get update
RUN apt-get install python3 -y
RUN apt-get install wget -y

CMD ["python3", "app.py"]