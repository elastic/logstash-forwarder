#ifndef _STRLIST_H_
#define _STRLIST_H_
typedef struct strlist {
  char **items;
  int nitems;
  int max_items;
} strlist_t;

strlist_t* strlist_new();
void strlist_free(strlist_t *list);
void strlist_append(strlist_t *list, const char *str);

void split(strlist_t **tokens, const char *buf, const char *sep);

#endif /* _STRLIST_H_ */
