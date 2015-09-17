/*
This simple program simulates a concurrent insert load into a mongodb collection. 

Part of the behavior is controlled by the following environment variables: 
- ITERATIONS
  Specifies how many insert iterations to execute (per thread). Defaults to 1000.  
- THREADS 
  Specifies the number of insert threads to start. Defaults to 2 
- MONGO
  Specifies the mongod/mongos instance to connect to. Defaults to mongodb://localhost:27017/
No sanity is performed on the variables so if you pass anything other than integers for the first two, and a proper mongodb URL for the third one, program behavior will be unpredictable (it is not even guaranteed to fail). 

*/

// I will use printf() from stdio
#include <stdio.h>
// I will use getenv() and rand() from stdlib
#include <stdlib.h>
// I will use time() from time
#include <time.h>
// these are the mongo-c required includes
//#include <bson.h>
#include <mongoc.h>
// I will use pthreads 
#include <pthread.h>

char *MONGO;

int _atoi(char *_var, int _default)
{
  return _var ? atoi(_var) : _default;
}

void *worker(void *_iterations)
{
  int *iterations = _iterations;
  mongoc_client_t *client;
  mongoc_collection_t *collection;
  //mongoc_cursor_t *cursor;
  bson_error_t error;
  bson_oid_t oid;
  bson_t *doc;

  mongoc_init();
  //client = mongoc_client_new("mongodb://localhost:27017/");
  client = mongoc_client_new(MONGO);
  collection = mongoc_client_get_collection(client, "metrics", "debug");
  
  for (int i=0; i<*iterations; i++) {
    doc = bson_new();
    bson_oid_init(&oid, NULL);
    BSON_APPEND_OID(doc, "_id", &oid);
    BSON_APPEND_TIME_T(doc, "ts", time(NULL));
    if (!mongoc_collection_insert(collection, MONGOC_INSERT_NONE, doc, NULL, &error)) {
      printf("%s\n", error.message);
    }
  }
  printf("\n");
  return NULL;
}

int main()
{
  char *_iterations = getenv("ITERATIONS");
  char *_threads = getenv("THREADS");
  MONGO = getenv("MONGO");
  if (MONGO == NULL) {
    MONGO = "mongodb://localhost:27017";
  }
    
  int iterations = _atoi(_iterations,1000);
  int threads = _atoi(_threads,2);
  pthread_t pthreads[threads];
  
  for (int thread = 0; thread < threads; thread++) {
    pthread_create(&pthreads[thread], NULL, worker, &iterations);
  }

  // wait for them all to finish
  for (int thread = 0; thread < threads; thread++) {
    pthread_join(pthreads[thread],NULL);
  }
 
 
}
