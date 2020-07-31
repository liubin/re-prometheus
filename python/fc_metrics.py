import json

def underbar_to_camel(s):
    ss = s.split("_")
    r=""
    for s in ss:
        r = r + s.capitalize()
    return r

def json_tag(s):
    return "`json:\""+s+"\"`"

metricDeclare="""{fn} = prometheus.NewGaugeVec(prometheus.GaugeOpts{{
        Namespace: namespaceFirecracker,
        Name:      "{tag}",
        Help:      "{help}",
    }},
        []string{{"item"}},
    )"""

def lowercase(s):
    if s == "RTCDeviceMetrics":
        return "rtcDeviceMetrics"
    return s[0].lower() + s[1:]

def uppercase(s):
    if s == "rtcDeviceMetrics":
        return "RTCDeviceMetrics"
    return s[0].upper() + s[1:]

def generate_declaration(fn, tag, help):
    fn=lowercase(fn)
    return metricDeclare.format(fn=fn, tag=tag, help=help)


type_name_tag_map={}

# only for FirecrackerMetrics struct
type_name_field_name_map={}
all_metrics=[]

def parse_metric_declaration():
    file1 = open('fc_metrics.txt', 'r') 
    Lines = file1.readlines() 
    # help: name: tag: items:{go_var: json_tag}
    metric={}
    in_struct=False
    in_FirecrackerMetrics=False
    for line in Lines: 
        line=line.rstrip()
        lline=line.lstrip()
        if lline.find("#[") > -1:
            continue

        if lline.find("///") > -1:
            if in_struct == False:
                in_struct=True
                hp=line[3:].strip()
                metric={"help": hp}
            continue
        if line == "}":
            in_struct=False
            in_FirecrackerMetrics=False
            all_metrics.append(metric)
            continue

        ss = line.split()
        if len(ss) == 4:
           ## pub struct MmdsMetrics {
           ## or type FirecrackerMetrics struct {
           type_name = ss[2]
           if ss[0] == "type":
               type_name = ss[1]
               in_FirecrackerMetrics=True
           metric["name"]=lowercase(type_name)
           mt=type_name_tag_map.get(type_name, "")
           if mt != "":
               metric["tag"]=mt

        elif len(ss) == 3:
            ## pub rx_accepted: SharedMetric,
            # mapped to metric's item label
            ## or pub patch_api_requests: PatchRequestsMetrics,
            nn = ss[1].rstrip(":")
            tt = ss[2].rstrip(",")
            if tt == "SharedMetric":
                tt ="uint64"
            else:
                type_name_tag_map[tt]=nn
            if in_FirecrackerMetrics:
            	type_name_field_name_map[tt]=underbar_to_camel(nn)

            items=metric.get("items",[])
            items.append((underbar_to_camel(nn),nn))
            metric["items"]=items

    # print(json.dumps(type_name_tag_map, indent=4))
    # print(json.dumps(all_metrics, indent=4))

declaration_stmt=[]
register_stmt=[]
set_metrics_stmt=[]
set_metrics_stmt_tpl="{metric_var}.WithLabelValues(\"{item}\").Set(float64({instance_var}.{field}))"

def generate_metric_declaration_codes():
    for m in all_metrics:
        metric_var_name=m["name"]
        if metric_var_name == "firecrackerMetrics":
            # the outer wrapper struct
            continue
        ds = generate_declaration(metric_var_name, m.get("tag","FIXME"), m["help"])
        declaration_stmt.append(ds)
        register_stmt.append("    prometheus.MustRegister("+metric_var_name+")")


        # set metrics
        set_metrics_stmt.append("")
        set_metrics_stmt.append("// set metrics for "+metric_var_name)
        for i in m.get("items",[]):
            iv="fm." + uppercase(type_name_field_name_map.get(uppercase(metric_var_name),"FIXME"))
            set_stmt = set_metrics_stmt_tpl.format(metric_var=metric_var_name,item=i[1],instance_var=iv,field=i[0])
            set_metrics_stmt.append(set_stmt)


def print_metric_declaration():

    print "var ("
    for x in declaration_stmt:
        print x
        print
    print ")"

    print
    print

    print "func registerFirecrackerMetrics() {"
    for x in register_stmt:
        print x
    print "}"

    print
    print


    print "func updateFirecrackerMetrics(fm *FirecrackerMetrics) {"
    for x in set_metrics_stmt:
        print x
    print "}"

parse_metric_declaration()
generate_metric_declaration_codes()
print_metric_declaration()

def translate_rust_to_go_types():
    file1 = open('fc_metrics.txt', 'r') 
    Lines = file1.readlines() 

    # Strips the newline character 
    for line in Lines: 
        line=line.rstrip()
        lline=line.lstrip()
        if lline.find("#[") > -1:
            continue

        if lline.find("//") > -1:
            print("{}".format(line))
            continue

        ss = line.split()
        if len(ss) == 4:
           ## pub struct MmdsMetrics {
           ##                               or 
           ## type FirecrackerMetrics struct {
           type_name = ss[2]
           if ss[0] == "type":
               type_name = ss[1]
           print("type {} struct {{".format(type_name))

        elif len(ss) == 3:
            ## pub rx_accepted: SharedMetric,
            nn = ss[1].rstrip(":")
            tt = ss[2].rstrip(",")
            if tt == "SharedMetric":
                tt ="uint64"
            print("   {} {} {}".format(underbar_to_camel(nn), tt, json_tag(nn)))

        else:
            print("{}".format(line))


translate_rust_to_go_types()