#include "strlist.h"
#define _BSD_SOURCE /* for stddup in glibc */
#include <stdlib.h>
#include <string.h>

strlist_t* strlist_new() {
  strlist_t *list;
  list = malloc(sizeof(strlist_t));

  list->max_items = 10;
  list->nitems = 0;
  list->items = malloc(list->max_items * sizeof(char*));

  return list;
}

void strlist_free(strlist_t *list) {
  int i;
  for (i = 0; i < list->nitems; i++)
    free(list->items[i]);
  free(list->items);
  free(list);
}

void strlist_append(strlist_t *list, const char *str) {
  list->items[list->nitems] = strdup(str);

  list->nitems++;
  if (list->nitems == list->max_items) {
    list->max_items *= 2;
    list->items = realloc(list->items, list->max_items * sizeof(char *));
  }
}

void split(strlist_t **tokens, const char *buf, const char *sep) {
  char *strptr = NULL;
  char *tokctx;
  char *dupbuf = NULL;
  char *tok;

  dupbuf = strdup(buf);
  strptr = dupbuf;

  *tokens = strlist_new();

  //printf("Split: '%s' on '%s'\n", buf, sep);
  while ((tok = strtok_r(strptr, sep, &tokctx)) != NULL) {
    strptr = NULL;
    strlist_append(*tokens, tok);
  }
  free(dupbuf);
}

