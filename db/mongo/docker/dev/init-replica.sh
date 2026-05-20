#!/bin/bash

# Wait for MongoDB to be ready to accept connections
until mongosh -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "print(\"waited for connection\")" > /dev/null 2>&1; do
  sleep 2
done

echo "MongoDB is ready. Initializing replica set 'rs0' for DEV..."

mongosh -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval '
  try {
    var status = rs.status();
    if (status.ok !== 1) {
      throw "Not initialized";
    }
    print("Replica set is already initialized.");
  } catch(e) {
    print("Initializing replica set...");
    rs.initiate({
      _id: "rs0",
      members: [{ _id: 0, host: "mongo_db:27017" }]
    });
  }
'
