import docker

class DockerAPI:
    def __init__(self):
        self.__client = docker.from_env()

    def list_kademlia_containers(self):
        containers = self.__client.containers.list()
        return [(container.name, list(container.ports.keys())[0][:-4], list(container.ports.values())[0][0]['HostPort']) for container in containers if container.image.tags[0] == "kademlia-peer:latest" and container.status != "exited"]