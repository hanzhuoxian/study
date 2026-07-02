#include <stdio.h>
#include <stdlib.h>
#include <setjmp.h>

#define TOK_ADD 5
#define MAXLINE 1000

jmp_buf jmpbuffer;

void do_line(char *);
void cmd_add(void);
int get_token(void);

int main(int argc, char const *argv[])
{
    char line[MAXLINE];
    if (setjmp(jmpbuffer) != 0)
    {
        printf("error\n");
    }
    while (fgets(line, MAXLINE, stdin) != NULL)
    {
        printf("do_line\n");
        do_line(line);
    }

    return 0;
}

/**
 * @brief 
 * global pointer for get_token()
 */
char *tok_ptr;

/**
 * @brief process one line fo input
 * 
 * @param ptr 
 */
void do_line(char *ptr)
{
    int cmd;

    tok_ptr = ptr;

    while ((cmd = get_token()) > 0)
    {
        printf("get_token : %d\n", cmd);
        switch (cmd)
        {
        case TOK_ADD:
            cmd_add();
            break;

        default:
            printf("get token default\n");
            longjmp(jmpbuffer, 1);
            break;
        }
        tok_ptr = "";
    }
}

void cmd_add(void)
{
    int token;
    printf("cmd_add\n");
    token = get_token();
    if (token < 0)
    {
        longjmp(jmpbuffer, 1);
    }
}

int get_token(void)
{
    char *t = &tok_ptr[0];
    return atoi(t);
}