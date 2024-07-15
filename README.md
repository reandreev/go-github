# go-github
GitHub REST API in Go

## Running

### Docker

```
docker pull reandreev/go-github
docker run -d -p 8080:8080 reandreev/go-github
```

### Minikube

```
kubectl apply -f kubernetes.yml
kubectl wait --for=condition=ready pod -l --app=go-github
minikube service go-github --url
```

## Authenticating

Use `/auth` with `POST` method providing to authenticate

```
curl -X POST -d '{"token": "<GITHUB_TOKEN>"}' localhost:8080/auth
```

## Listing repositories

Use `/repos` with `GET` method to get a list of your own repos or `/repos/<user>` to get a list of public repos owned by `<user>`

```
curl localhost:8080/repos
curl localhost:8080/repos/torvalds
```

## Creating repositories

Use `/repos` with `POST` method to create a new repository

```
curl -X POST -d '{"name": "<REPOSITORY_NAME>"}' localhost:8080/repos
```

## Deleting repositories

Use `/repos/<user>/<repo>` with `DELETE` method to delete a repository

```
curl -X DELETE localhost:8080/repos/reandreev/go-github
```

## Listing pull requests

Use `/pulls/<user>/<repo>/<n>` with `GET` method to list the `<n>` latest open pull requests in `/<user>/<repo>`

```
curl localhost:8080/pulls/torvalds/linux/5
```

## Logging out

Use `/logout` with `GET` method to logout

```
curl localhost:8080/logout
```

## TODO  
- [X] create REST API that allows create, destroy, and list repositories in github  
- [X] create REST API that allows for a certain repo list the N pull requests open  
- [X] deployment is done on minikube  
- [X] pipeline for running tests, lint, security check and finally deploy  
