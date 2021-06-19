export PROJECT_ID=jjgames
export TAG=v1
docker build -f Dockerfile -t gcr.io/$PROJECT_ID/sr-games-backend:$TAG .
gcloud services enable containerregistry.googleapis.com
gcloud auth configure-docker
docker push gcr.io/$PROJECT_ID/sr-games-backend:$TAG
read