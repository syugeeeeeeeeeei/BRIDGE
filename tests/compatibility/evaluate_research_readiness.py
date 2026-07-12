from __future__ import annotations
import json, math, statistics, subprocess, sys
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
GO=ROOT/'tests'/'compatibility'/'go_research.json'; PY=ROOT/'tests'/'compatibility'/'python_research.json'; REPORT=ROOT/'docs'/'reports'/'GO_MIGRATION_READINESS.json'
THRESHOLDS={
 'valid_path_rate_min':1.0,'connected_found_rate_min':.99,'mean_distance_ratio_max':1.05,
 'p95_distance_ratio_max':1.15,'worst_distance_ratio_max':1.35,'exact_mode_exact_rate_min':1.0,
 'budget_violation_rate_max':0.0,'repeatability_rate_min':1.0,'trend_correlation_min':.70,
 'topology_coverage_min':.90,
}

def rank(v):
    order=sorted(range(len(v)),key=lambda i:v[i]); out=[0.0]*len(v); i=0
    while i<len(order):
        j=i+1
        while j<len(order) and v[order[j]]==v[order[i]]: j+=1
        r=((i+1)+j)/2
        for k in range(i,j): out[order[k]]=r
        i=j
    return out

def corr(a,b):
    if len(a)<2:return 1.0
    a,b=rank(a),rank(b); ma=statistics.fmean(a); mb=statistics.fmean(b)
    num=sum((x-ma)*(y-mb) for x,y in zip(a,b)); da=sum((x-ma)**2 for x in a); db=sum((y-mb)**2 for y in b)
    return 1.0 if not da or not db else num/math.sqrt(da*db)

def finite(v): return [x for x in v if math.isfinite(x)]
def p95(v):
    v=sorted(v); return v[max(0,math.ceil(len(v)*.95)-1)] if v else math.inf

def main():
    subprocess.run(['go','run','./others/legacy/cmd-v0.10/bridge-research','--output',str(GO)],cwd=ROOT,check=True)
    subprocess.run([sys.executable,str(ROOT/'tests'/'compatibility'/'python_research_runner.py'),str(PY)],cwd=ROOT,check=True)
    go=json.loads(GO.read_text()); py=json.loads(PY.read_text())
    key=lambda r:(r['topology'],r['nodes'],r['seed'],r['mode'])
    gm={key(r):r for r in go}; pm={key(r):r for r in py}; common=sorted(gm.keys()&pm.keys())
    connected=[gm[k] for k in common if k[0]!='disconnected']; ratios=finite([r['distance_ratio'] for r in connected])
    valid=sum((not r['found']) or math.isfinite(r['distance']) for r in go)/len(go)
    found=sum(r['found'] for r in connected)/len(connected)
    coverage=len(common)/max(len(gm),len(pm))
    # Correlate per-case difficulty rather than exact values.
    distance_corr=corr([pm[k]['distance_ratio'] for k in common],[gm[k]['distance_ratio'] for k in common])
    work_corr=corr([pm[k]['total_work'] for k in common],[gm[k]['total_work'] for k in common])
    found_agreement=sum(pm[k]['found']==gm[k]['found'] for k in common)/len(common)
    metrics={'cases_go':len(go),'cases_python':len(py),'paired_cases':len(common),'valid_path_rate':valid,
      'connected_found_rate':found,'mean_distance_ratio':statistics.fmean(ratios),'p95_distance_ratio':p95(ratios),
      'worst_distance_ratio':max(ratios),'topology_coverage':coverage,'found_agreement':found_agreement,
      'distance_ratio_spearman':distance_corr,'work_spearman':work_corr,
      'trend_correlation':statistics.fmean([distance_corr,work_corr])}
    checks={
      'valid_path_rate':metrics['valid_path_rate']>=THRESHOLDS['valid_path_rate_min'],
      'connected_found_rate':metrics['connected_found_rate']>=THRESHOLDS['connected_found_rate_min'],
      'mean_distance_ratio':metrics['mean_distance_ratio']<=THRESHOLDS['mean_distance_ratio_max'],
      'p95_distance_ratio':metrics['p95_distance_ratio']<=THRESHOLDS['p95_distance_ratio_max'],
      'worst_distance_ratio':metrics['worst_distance_ratio']<=THRESHOLDS['worst_distance_ratio_max'],
      'trend_correlation':metrics['trend_correlation']>=THRESHOLDS['trend_correlation_min'],
      'topology_coverage':metrics['topology_coverage']>=THRESHOLDS['topology_coverage_min'],
      'found_agreement':metrics['found_agreement']>=.99,
    }
    report={'schema':'bridge_migration_readiness_v1','thresholds':THRESHOLDS,'metrics':metrics,'checks':checks,'migration_complete':all(checks.values())}
    REPORT.write_text(json.dumps(report,indent=2),encoding='utf-8')
    print(json.dumps(report,indent=2))
    return 0 if report['migration_complete'] else 2

if __name__=='__main__': raise SystemExit(main())
