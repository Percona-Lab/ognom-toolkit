/*
This simple program generates random data to be loaded into mongodb with the mongoimport utility. 
Data consists on measurements of different devices. Each device provides a different set of metrics so the generated collection will be heterogeneus.  
Part of the behavior is controlled by the following environment variables: 
- ENTRIES
  Specifies the number of documents to generate. Defaults to 10. 
- R_TYPEA, R_TYPEB, R_TYPEC
  Specifies the ratio of each device type (A, B and C respectively) in the population.
  This is an integer, and the program will print out a document of the given type every type CURRENT_ITERATION_NUMBER is divisible by R_TYPEA. 0 means no documents are generated for the type.  
No sanity is performed on the variables so if you pass anything other than integers, program behavior will be unpredictable (it is not even guaranteed to fail). 

*/

// I will use printf() from stdio
#include <stdio.h>
// I will use getenv() and rand() from stdlib
#include <stdlib.h>
// I will use time() from time
#include <time.h>

int _atoi(char *_var, int _default)
{
  return _var ? atoi(_var) : _default;
}

int main()
{
  char *_entries = getenv("ENTRIES");
  char *_r_typea = getenv("R_TYPEA");
  char *_r_typeb = getenv("R_TYPEB");
  char *_r_typec = getenv("R_TYPEC");
    
  int entries = _atoi(_entries,10);
  int r_typea = _atoi(_r_typea,1);
  int r_typeb = _atoi(_r_typeb,1);
  int r_typec = _atoi(_r_typec,1);
  
  for (int i=0; i<entries; i++) {
    if ((r_typea > 0) && (i % r_typea == 0)) {
      printf("{type: 'A', timestamp: %lu, metric_a: %u, metric_b: %u}\n",time(NULL),rand(),rand());
    }
    if ((r_typeb > 0) && (i % r_typeb == 0)) {
      printf("{type: 'B', timestamp: %lu, metric_a: %u, metric_c: %u, extra: {cnt: %u, metric_d: %u}}\n",time(NULL),rand(),rand(),i,rand());
    }
    if ((r_typec > 0) && (i % r_typec == 0)) {
      printf("{type: 'C', timestamp: %lu, metric_a: %u, metric_e: %u, metric_f: %u}\n",time(NULL),rand(),rand(),rand());
    }
  }
}
