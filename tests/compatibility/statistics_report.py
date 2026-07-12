from __future__ import annotations
import argparse, csv, json, math, random, statistics
from pathlib import Path

def cliffs_delta(a,b):
    if not a or not b: return math.nan
    gt=sum(x>y for x in a for y in b); lt=sum(x<y for x in a for y in b)
    return (gt-lt)/(len(a)*len(b))

def bootstrap_ci(values, seed=1, samples=2000, alpha=.05):
    if not values: return [math.nan,math.nan]
    r=random.Random(seed); means=sorted(statistics.fmean(r.choices(values,k=len(values))) for _ in range(samples))
    return [means[int(samples*alpha/2)],means[min(samples-1,int(samples*(1-alpha/2)))]]

def mann_whitney_u(a,b):
    pairs=sorted([(x,0) for x in a]+[(x,1) for x in b]); ranks=[0.0]*len(pairs); i=0
    while i<len(pairs):
        j=i+1
        while j<len(pairs) and pairs[j][0]==pairs[i][0]: j+=1
        rank=(i+1+j)/2
        for k in range(i,j): ranks[k]=rank
        i=j
    r1=sum(r for r,p in zip(ranks,pairs) if p[1]==0); u1=r1-len(a)*(len(a)+1)/2; u2=len(a)*len(b)-u1
    u=min(u1,u2); mu=len(a)*len(b)/2; sigma=math.sqrt(len(a)*len(b)*(len(a)+len(b)+1)/12)
    z=0 if sigma==0 else (u-mu+.5)/sigma; p=math.erfc(abs(z)/math.sqrt(2))
    return {'u':u,'z':z,'p_value_approx':p}

def summarize(v):
    return {'n':len(v),'mean':statistics.fmean(v) if v else math.nan,'median':statistics.median(v) if v else math.nan,'stddev':statistics.stdev(v) if len(v)>1 else 0.0,'bootstrap_ci95':bootstrap_ci(v)}

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('input'); ap.add_argument('--metric',default='end_to_end_time_ms'); ap.add_argument('--group',default='algorithm'); ap.add_argument('-o','--output'); args=ap.parse_args()
    data=json.loads(Path(args.input).read_text()); rows=[r for r in data['raw_runs'] if not r.get('warmup')]
    groups={}
    for r in rows: groups.setdefault(str(r[args.group]),[]).append(float(r[args.metric]))
    report={'metric':args.metric,'group':args.group,'groups':{k:summarize(v) for k,v in groups.items()},'comparisons':[]}
    keys=sorted(groups)
    for i,a in enumerate(keys):
        for b in keys[i+1:]: report['comparisons'].append({'a':a,'b':b,'cliffs_delta':cliffs_delta(groups[a],groups[b]),'mann_whitney_u':mann_whitney_u(groups[a],groups[b])})
    text=json.dumps(report,indent=2,allow_nan=False); Path(args.output).write_text(text+'\n') if args.output else print(text)
if __name__=='__main__': main()
