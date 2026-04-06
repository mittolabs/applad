"""Storage service."""


class Storage:
    def __init__(self, client):
        self.client = client

    def create_bucket(self, name: str):
        return self.client._call("POST", "/storage/buckets", {
            "name": name,
            "bucketId": "unique()",
            "permissions": [],
            "allowedFileExtensions": [],
            "encryption": False,
            "antivirus": False,
        })

    def list_buckets(self):
        return self.client._call("GET", "/storage/buckets")

    def list_files(self, bucket_id: str):
        return self.client._call("GET", f"/storage/buckets/{bucket_id}/files")

    def get_file(self, bucket_id: str, file_id: str):
        return self.client._call("GET", f"/storage/buckets/{bucket_id}/files/{file_id}")

    def delete_file(self, bucket_id: str, file_id: str):
        return self.client._call("DELETE", f"/storage/buckets/{bucket_id}/files/{file_id}")
