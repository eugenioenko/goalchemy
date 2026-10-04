#include "goalchemy.h"
#include <assert.h>
#include <math.h>
#include <stdio.h>

static gxc_value number(double value) { return (gxc_value){.kind=GXC_FLOAT,.floating=value}; }
static gxc_value call(const char *name, gxc_value *input, size_t count) {
    gxc_value result={0}; gxc_error error={0};
    int status=goalchemy_invoke(name,input,count,NULL,&result,&error);
    if(status) fprintf(stderr,"%s failed kind %d\n",name,status);
    assert(status==0); gxc_error_free(&error); return result;
}
int main(void) {
    double values[]={1.25,-0.0,INFINITY,-INFINITY,NAN};
    for(size_t i=0;i<5;i++) {
        gxc_value input=number(values[i]);
        gxc_value small=call("Scalar32",&input,1),wide=call("Scalar64",&input,1);
        assert(small.kind==GXC_FLOAT && wide.kind==GXC_FLOAT);
        if(isnan(values[i])) { assert(isnan(small.floating) && isnan(wide.floating)); }
        else { assert(small.floating==(double)(float)values[i] && wide.floating==values[i]); assert(!!signbit(small.floating)==!!signbit(values[i]) && !!signbit(wide.floating)==!!signbit(values[i])); }
        gxc_value_free(&small);gxc_value_free(&wide);
    }
    gxc_value pair[]={number(1.25),number(-0.0)},both=call("Both",pair,2);
    assert(both.kind==GXC_LIST && both.length==2 && both.items[0].kind==GXC_FLOAT && both.items[0].floating==1.25 && both.items[1].kind==GXC_FLOAT && signbit(both.items[1].floating));gxc_value_free(&both);
    gxc_value rounded=number(16777217.0),result=call("Scalar32",&rounded,1);
    assert(result.floating==16777216.0);gxc_value_free(&result);
    gxc_value listitems[]={number(2.5)},inneritems[]={number(4.5)},pairitems[]={number(6.5),number(7.5)};
    gxc_value inner={.kind=GXC_LIST,.items=inneritems,.length=1};
    gxc_value fields[]={number(1.25),number(-0.0),{.kind=GXC_LIST,.items=listitems,.length=1},{.kind=GXC_LIST,.items=&inner,.length=1},{.kind=GXC_LIST,.items=pairitems,.length=2}};
    char *names[]={"Small","Wide","Values","Nested","Pair"};
    gxc_value input={.kind=GXC_RECORD,.items=fields,.names=names,.length=5};
    result=call("Echo",&input,1);
    assert(gxc_field(&result,"Small")->floating==1.25 && signbit(gxc_field(&result,"Wide")->floating));
    assert(gxc_field(&result,"Values")->items[0].floating==3.5 && gxc_field(&result,"Nested")->items[0].items[0].floating==6.5 && gxc_field(&result,"Pair")->items[1].floating==7.5);
    assert(listitems[0].floating==2.5 && inneritems[0].floating==4.5);
    gxc_value *owneditems=gxc_field(&result,"Values")->items;owneditems[0].floating=99;
    gxc_value again=call("Echo",&input,1);assert(gxc_field(&again,"Values")->items[0].floating==3.5 && owneditems[0].floating==99);gxc_value_free(&again);gxc_value_free(&result);
    result=call("Suspended",&input,1);assert(gxc_field(&result,"Values")->items[0].floating==2.5);gxc_value_free(&result);
    result=call("Fixed",NULL,0);owneditems=gxc_field(&result,"Values")->items;owneditems[0].floating=99;again=call("Fixed",NULL,0);assert(gxc_field(&again,"Values")->items[0].floating==1.5 && owneditems[0].floating==99);gxc_value_free(&again);gxc_value_free(&result);
    gxc_error error={0};gxc_value bad={.kind=GXC_BOOL,.integer=1};
    assert(goalchemy_invoke("Scalar32",&bad,1,NULL,&result,&error)==5);gxc_error_free(&error);
    fields[4].length=1;assert(goalchemy_invoke("Echo",&input,1,NULL,&result,&error)==5);gxc_error_free(&error);
    puts("PASS float library");return 0;
}
