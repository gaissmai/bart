\datethis
@s __inline__ extern @q reserve a gcc keyword @>
@s bool normal @q unreserve a C++ keyword @>

@* IP route lookup. This program is a sketch of the ``allotment routing
table'' method for selecting the longest prefix match, given
a dynamically changing set of prefixes.

I should take time to describe the ideas carefully, but I don't have much time
today --- so I'll assume that the reader is very SMART.

Another thing I don't have time to do is make this program of
industrial-strength quality. Sorry! But maybe it will still be of interest.

This \.{CWEB} source produces a header file \.{iprt.h} as well as
a \CEE/ program \.{iprt.c}, so that other programs can easily
interface with it. But I'm not sure I have put all the necessary
stuff into the header properly.

The present module defines subroutines that the main routine
will call. I'm keeping it as simple as I can. Unfortunately I don't
have time to write a substantial main routine, or to make thorough
checks, although I didn't find any bugs during rudimentary testing.

@c
#include "iprt.h"
@<Global variables@>@;
@<Private prototypes@>@;
@<Inline subroutines@>@;
@<Subroutines@>@;

@ @(iprt.h@>=
#include <malloc.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
@<Type definitions@>@;
@<External declarations@>@;
@<Public prototypes@>@;

@* Basic data structures. Routes are assumed to be stored in a
doubly linked list. The entries of this list contain a destination
address and a mask, specifying a prefix address: If the prefix
is a $k$-bit binary string, the destination address will be
that string shifted left $32-k$ bits, and the mask will be
$-2^{32-k}$. For example, a prefix of $(0101011101)_2$ has $k=10$,
so it is represented by the destination address |0x57400000| and
the mask |0xffc00000|.

I like to think of the prefix 0101011101 also as the string
0101011101 followed by 22 asterisks. Or, we can regard any
prefix as the positive binary number obtained by prepending a~1;
then 0101011101 corresponds to the binary number $(10101011101)_2$.
The null prefix corresponds to 1, and all prefixes of up to 32 bits
correspond to the binary numbers from 1 to $(11\ldots1)_2=2^{33}-1$.

The destination and mask are unsigned 32-bit integers. In this program
they're said to be of type \&{ipv4a}, for ``Internet Protocol Version~4
Address.''

Elements of the linked list must also contain other information
(like the ``next hop address'' for reaching the destination),
but I'm leaving that out here for simplicity. Instead, I only
include a short string identifier for use in diagnostic printouts.

@<Type ...@>=
typedef unsigned int ipv4a, u32; /* we assume 32-bit arithmetic */
typedef struct route {
  struct route *next, *prev; /* double linking */
  ipv4a dest, mask; /* prefix defining this route */
  char id[8]; /* identifier, stands for other info that is suppressed here */
} route;

@ Prefixes of up to 16 bits are handled in a first-level table
with $(11\ldots1)_2=2^{17}-1$ entries. We never have to update more
than 511 of those entries at a time, because non-null prefixes always
have length at least 8.

The first-level table entry for each prefix points to the longest route
that matches it. The null route is always assumed to be present; thus,
even if no non-null route is consistent with a given prefix, the null
route always does match.

Prefixes of more than 16 bits are handled in tables on the second or third
level. When such prefixes are present, say a prefix of length $>16$ whose
first sixteen bits are the string $\alpha$, we put a link to the
second-level table in position $(1\alpha)_2$ of the first-level table.
In such a case we don't have room to point to the longest prefix that
leads to $\alpha$, so we put that link into position~1 of the
second-level table. Such a link is called the ``base pointer'' at the
next level; it might correspond to a prefix of length 16 even though
it appears in the second level.

Thus the table entries are from a union type, either a route pointer or
a pointer to another table. We will distinguish between them by adding~1 to
the table pointers, knowing that ordinary pointers always are even.

The table entries in second-level or third-level tables are like those
on the first level, except that they may have null pointers.
A null pointer (|NULL|) stands for the value of the base pointer in its table.
A prefix of length 27, having the binary form $\alpha\beta\gamma$ where
$\alpha$ has length~16 and $\beta$ has length~8 and $\gamma$ has length~3,
corresponds to the route in entry $(1\gamma)_2$ of the subtable pointed to
in entry $(1\beta)_2$ of the subtable pointed to in entry $(1\alpha)_2$
of the first-level table.

Position 0 of a subtable is used to count the number of active
elements in the subtable (either routes or pointers to a deeper level).
We therefore allow unsigned integers to part of the same union type.

@<Type...@>=
typedef union table_entry{
    route *ent;
    union table_entry *down;
    u32 count;
} table_entry;
typedef table_entry *subtable;
@#
#define is_subtable(p) (p.count&1)
#define make_subtable(p) @,@,@[(subtable)((u32)(p)|1)@]
#define subtable_ptr(p) @,@,@[((table_entry)(p.count&-2))@]

@ The first-level table, called |root_table|, is allocated permanently.

@<Extern...@>=
extern table_entry root_table[1<<17];

@ @<Glob...@>=
table_entry root_table[1<<17];

@ We also use traditional Boolean conventions.

@<Type...@>=
typedef enum{@+false,true@+} bool;

@* Lookup. We can get familiar with these data structures by looking
at the |find_match| routine, which finds the longest existing prefix
that matches a given 32-bit address. This is the bread-and-butter
subroutine, the one we want to be blazingly fast.

@<Public...@>=
route* find_match(ipv4a);

@ @<Sub...@>=
route* find_match(ipa)
  register ipv4a ipa;
{
  register table_entry x,y,z;
  x=root_table[(ipa>>16)+(1<<16)];
  if (!is_subtable(x)) return x.ent;
  x=subtable_ptr(x);
  y=x.down[((ipa>>8)&0xff)+(1<<8)];
  if (!is_subtable(y)) return y.ent? y.ent: x.down[1].ent;
  y=subtable_ptr(y);
  z=y.down[(ipa&0xff)+(1<<8)];
  return z.ent? z.ent: y.down[1].ent? y.down[1].ent: x.down[1].ent;
}

@* Initialization. At the beginning there is one route (for the null prefix),
and the first-level table entries all point to it.

@<Glob...@>=
route null_route;

@ @<Public...@>=
void initialize(void);

@ @<Sub...@>=
void initialize()
{
  register int i;
  null_route.next=null_route.prev=&null_route;
  null_route.dest=null_route.mask=0;
  strcpy(null_route.id,"NULL");
  for (i=1; i<1<<17; i++)
    root_table[i].ent=&null_route;
}

@* Subroutines for updates. We need a few utility routines, which are
collected here: |mask_length| computes the number of 1 bits in a mask,
|new_route| allocates a new route, |free_route| deallocates a given route,
|new_subtable| allocates a new subtable having a given base,
and |free_subtable| frees a subtable returning its base.

@<Private...@>=
int mask_length(ipv4a);
route *new_route(ipv4a,ipv4a,char*);
void free_route(route*);
subtable new_subtable(table_entry);
table_entry free_subtable(subtable);

@ A simple loop will compute the mask length, but for speed it's more fun
to do something tricky. Here I implement it in two ways: One uses standard
IEEE floating point arithmetic, and the other uses division by 37.
Perhaps both of these take longer than the simple loop, on some computers,
but that remains to be tested.

@<Sub...@>=
#ifdef IEEE_floating_point
int mask_length(mask)
  ipv4a mask;
{
  register union {@+float f;@+ u32 u;@+}@+ x;
  x.f=(float)(-mask);
  return 0x9f-(x.u>>23);
}
#else
int magic[37]={ 0,  5,  9, 30,  6, 31, 32,  0, 14, 13,
               24, 12, 27, 23, 18, 11,  0, 26, 20, 22,
                3, 17,  1, 10,  7,  0, 15, 25, 28, 19,
                0, 21,  4,  2,  8, 16, 29};
__inline__ int mask_length(mask)
  ipv4a mask;
{
  return magic[mask%37];
}
#endif

@ @<Sub...@>=
route *new_route(dest,mask,id)
  register ipv4a dest,mask;
  char *id;
{
  register route* r;
  r=(route*)calloc(1,sizeof(route));
  if (!r) {
    fprintf(stderr,"Out of memory!\n");
    exit(-1);
  }
  r->dest=dest;@+ r->mask=mask;
  strncpy(r->id,id,7);
  r->next=null_route.next;@+ r->prev=&null_route;
  null_route.next=r->next->prev=r;
  return r;
}

@ @<Sub...@>=
void free_route(r)
  register route *r;
{ 
  register route *s,*t;
  if (!r) {
    fprintf(stderr,"Can't delete a NULL route!\n");
    exit(-2);
  }
  if (r==&null_route) {
    fprintf(stderr,"Can't delete the null route!\n");
    exit(-3);
  }
  s=r->prev, t=r->next;
  s->next=t, t->prev=s;
  free((void*)r);
}

@ @<Sub...@>=
subtable new_subtable(base)
  register table_entry base;
{
  register subtable t;
  t=(subtable)calloc(512,sizeof(table_entry));
  if (!t) {
    fprintf(stderr,"Out of memory!\n");
    exit(-4);
  }
  t[1]=base;
  return t;
}

@ @<Sub...@>=   
table_entry free_subtable(t)
  subtable t;
{
  register table_entry base;
  if (!t) {
    fprintf(stderr,"Can't delete a NULL subtable!\n");
    exit(-5);
  }  
  base=t[1];
  free((void*)t);
  return base;
}

@* Tree walking. The table at each level is essentially a complete
binary tree, as discussed for example in Section 2.3.4.5 of
{\sl Fundamental Algorithms}. The principal problem that we face when updating
the tree is to change a given table entry |t[k]|
from |r| to |s|; furthermore, if |k| isn't at the fringe of the
tree, we'll want to do the same thing recursively to table entries
|t[2*k]| and |t[2*k+1]|, if those table entries equal~|r|.

@<Priv...@>=
__inline__ void change(subtable,int,route*,route*,int,bool);

@ The program here avoids recursion overhead by using old-fashioned
methods that some people think are unstructured. This allows us to
optimize by using inline code.

The deepest entries of a tree are called its fringe. On level~1, the
fringe starts at position |1<<16|; on the other levels it starts at position
|1<<8|. On level~3 we do not have to worry about fringe elements pointing to
subtables, so the processing is faster.

The |change| procedure is called only when |k<threshold|. In other
words, the starting place |k| is never part of the fringe.

@<Inline subroutines@>=
__inline__ void change(t,k,r,s,threshold,fringe_check)
  subtable t; /* the table we're changing */
  int k; /* the place we start */
  route *r,*s; /* route |r| will be changed to route |s| */
  int threshold; /* where the fringe begins */
  bool fringe_check; /* can fringe nodes be subtables? */
{
  register int j=k;
start_change: j<<=1;
  if (j<threshold) goto non_fringe;
  while (1) {
    @<Change the fringe element |t[j]| if it matches |r|@>;
    if (j&1) goto move_up;
    j++;
  } /* the loop is executed exactly twice, so it could be optimized slightly */
non_fringe: if (t[j].ent==r) goto start_change;
move_on: if (j&1) goto move_up;
  j++;@+ goto non_fringe;
move_up: j>>=1;
  t[j].ent=s;
  if (j!=k) goto move_on;
}  

@ Here we deal with a slight complication due to the fact that
the route at a fringe element might be stored as the base of
another subtable, not in table~|t| itself.

@<Change the fringe element |t[j]| if it matches |r|@>=
if (fringe_check && is_subtable(t[j])) {
  if (subtable_ptr(t[j]).down[1].ent==r) subtable_ptr(t[j]).down[1].ent=s;
}@+ else if (t[j].ent==r) t[j].ent=s;

@* Inserting a new route. Now suppose we want to contribute a new
route to the data structures. The subroutine |insert_route| does this;
however, if it discovers that the route is already present, it simply
returns |false|.

@<Public...@>=
bool insert_route(ipv4a,ipv4a,char*);

@ @<Sub...@>=
bool insert_route(dest,mask,id)
  ipv4a dest,mask;
  char *id;
{
  register int k;
  register table_entry x,y;
  if (!mask) return false; /* null route is already present */
  k=mask_length(mask);
  if (k<8) {
    fprintf(stderr,"Can't insert a route of length %d!\n",k);
    exit(-6);
  }
  if (k<=16)
    return insert(root_table,(dest>>(32-k))+(1<<k),1<<16,true,dest,mask,id);
  x=root_table[(dest>>16)+(1<<16)];
  if (is_subtable(x)) x=subtable_ptr(x);
  else {
    x.down=new_subtable(x);
    root_table[(dest>>16)+(1<<16)].down=make_subtable(x.down);
  }
  if (k<=24)
    return insert(x.down,(((dest>>8)&0xff)>>(24-k))+(1<<(k-16)),
              1<<8,true,dest,mask,id);
  y=x.down[((dest>>8)&0xff)+(1<<8)];
  if (is_subtable(y)) y=subtable_ptr(y);
  else {
    y.down=new_subtable(y);
    x.down[((dest>>8)&0xff)+(1<<8)].down=make_subtable(y.down);
    x.down[0].count++;
  }
  return insert(y.down,((dest&0xff)>>(32-k))+(1<<(k-24)),
            1<<8,false,dest,mask,id);
}

@ The main work of insertion is handled by an inline routine, which takes
advantage of simplifications that occur at levels 1 and~3.

@<Priv...@>=
__inline__ bool insert(subtable,int,int,bool,ipv4a,ipv4a,char*);

@ Tests like `|threshold==1<<8|' are actually made at compile time,
not at run time, because the following code is compiled inline.
Therefore these extra tests actually make the program run faster,
although without inline compilation they would make it slower.
The value of |fringe_check| never actually appears in a runtime
register, it merely controls the selection of optional code.

@<Inline...@>=
__inline__ bool insert(t,k,threshold,fringe_check,dest,mask,id)
  subtable t; /* the table where insertion happens */
  int k; /* the place where it is launched */
  int threshold; /* where the fringe begins */
  bool fringe_check; /* can fringe entries point to subtables? */
  ipv4a dest,mask; /* the route to insert */
  char *id; /* the route identifier */
{
  register table_entry z=t[k];
  register route *r,*s;
  r=(fringe_check && is_subtable(z))? subtable_ptr(z).down[1].ent : z.ent;
  if ((threshold!=1<<8 || r) && r->dest==dest && r->mask==mask) return false;
  s=new_route(dest,mask,id);
  if (threshold==1<<8) t[0].count++;
  if (k<threshold) change(t,k,r,s,threshold,fringe_check);
  else if (fringe_check && is_subtable(z)) subtable_ptr(z).down[1].ent=s;
  else z.ent=s;
  return true;
}

@* Deleting a route. Conversely, we can downdate the data structures when
a route is supposed to be forgotten. The |delete_route| subroutine
returns |false| if the route isn't already present, or if it is null.

@<Public...@>=
bool delete_route(ipv4a,ipv4a);

@ @<Sub...@>=
bool delete_route(dest,mask)
  ipv4a dest,mask;
{
  register int k;
  register table_entry x,y;
  register table_entry *xx,*yy;
  if (!mask) return false; /* the null route cannot be deleted */
  k=mask_length(mask);
  if (k<=16)
    return delete(root_table,(dest>>(32-k))+(1<<k),
           1<<16,true,dest,mask,NULL,NULL,NULL);
  xx=&root_table[(dest>>16)+(1<<16)], x=*xx;
  if (is_subtable(x)) x=subtable_ptr(x);
  else return false;
  if (k<=24)
    return delete(x.down,(((dest>>8)&0xff)>>(24-k))+(1<<(k-16)),
              1<<8,true,dest,mask,xx,NULL,NULL);
  yy=&x.down[((dest>>8)&0xff)+(1<<8)], y=*yy;
  if (is_subtable(y)) y=subtable_ptr(y);
  else return false;
  return delete(y.down,((dest&0xff)>>(32-k))+(1<<(k-24)),
              1<<8,false,dest,mask,yy,xx,x.down);
}

@ The main work of deletion is handled by an inline routine, which takes
advantage of simplifications that occur at levels 1 and~3.

@<Priv...@>=
__inline__ bool
 delete(subtable,int,int,bool,ipv4a,ipv4a,table_entry*,table_entry*,subtable);

@ As before, we are happy that tests like `|threshold==1<<8|'
are actually made at compile time, not at run time.

One slightly tricky precaution needs to be taken if |k| is 2 or~3: Deleting
a route of length 17 or 25 should not reset the table entries to the
value of the base pointer of their subtable (although the base pointer
does point to the currently longest match after deletion). We must rather
reset the table entries to |NULL|, because the base pointer might change
at any time.

@<Inline...@>=
__inline__ bool
  delete(t,k,threshold,fringe_check,dest,mask,parent,grandparent,parent_table)
  subtable t; /* the table where deletion happens */
  int k; /* the place where it is launched */
  int threshold; /* where the fringe begins */
  bool fringe_check; /* can fringe entries point to subtables? */
  ipv4a dest, mask; /* the route to delete */
  table_entry *parent, *grandparent; /* subtable pointers that might go away */
  subtable parent_table; /* a parent subtable that might also go away */
{
  register table_entry z=t[k];
  register route *r,*s;
  r=(fringe_check && is_subtable(z))? subtable_ptr(z).down[1].ent : z.ent;
  if ((threshold==1<<8 && !r) || r->dest!=dest || r->mask!=mask) return false;
  free_route(r);
  s=(k>>1)>1? t[k>>1].ent: NULL; /* see the comment above */
  if (threshold==1<<8) {
    t[0].count--;
    if (!t[0].count) @<Free the subtable and |return|@>;
  }
  if (k<threshold) change(t,k,r,s,threshold,fringe_check);
  else if (fringe_check && is_subtable(z)) subtable_ptr(z).down[1].ent=s;
  else z.ent=s;
  return true;
}

@ When the last route of a subtable is deleted, we remove the subtable
as if it never existed. Removing a subtable at level~3 might, in turn,
lead to the removal of a subtable at level~2.

@<Free the subtable and |return|@>=
{
  r=free_subtable(t).ent;
  (*parent).ent=r;
  if (!fringe_check) {
    parent_table[0].count--;
    if (!parent_table[0].count) {
      r=free_subtable(parent_table).ent;
      (*grandparent).ent=r;
    }
  }
  return true;
}

@* Diagnostic routines. Here are a few utility routines that should
be useful for debugging.

@<Public...@>=
void print_routes(void);
void print_entry(table_entry);
void print_stats(subtable);
void print_entries(subtable,int,int);

@ @<Sub...@>=
void print_routes()
{
  register route *r;
  for (r=null_route.next; r!=&null_route; r=r->next)
    printf(" %s (%x,%d)\n",r->id,r->dest,mask_length(r->mask));
}

@ @<Sub...@>=
void print_entry(x)
  table_entry x;
{
  register route *r;
  register subtable xx;
  xx=is_subtable(x)? subtable_ptr(x).down: NULL;
  r=is_subtable(x)? xx[1].ent: x.ent;
  printf("%s", r? r->id: "(nil)");
  if (is_subtable(x)) printf(", subtable 0x%x",(u32)xx);
}
    
@ @<Sub...@>=
void print_stats(t)
  subtable t;
{
  if (!t) printf("Null subtable!");
  else {
    printf("Subtable has %d active elements, base pointer ",t[0].count);
    print_entry(t[1]);
  }
}

@ @<Sub...@>=
void print_entries(t,k,levels)
  subtable t;
  int k; /* where to start */
  int levels; /* how much further to descend */
{
  register int i,j,s;
  for (j=k,s=1; levels; j<<=1, s<<=1, levels--)
    for (i=0; i<s; i++) {
      printf("%6d: ",j+i);
      print_entry(t[j+i]);
      printf("\n");
    }
}

@* Index.
