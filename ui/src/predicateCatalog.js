export function predicateRows(structure) {
  const expansions=structure?.predicateExpansions??[];
  const rows=[];
  for(const declaration of structure?.declarations??[]){
    const parameter=declaration?.parameter;
    if(!parameter||String(parameter.source?.kind||'').toLowerCase()==='output')continue;
    for(const occurrence of declaration.predicates??[]){
      const predicate=occurrence?.predicate;
      if(!predicate)continue;
      const group=Number.isInteger(predicate.group)?predicate.group:0;
      const normalizedPredicate={...predicate,group};
      const groupExpansions=expansions.filter((item)=>item.group===group);
      const expansionViews=[...new Set(groupExpansions.map((item)=>item.view).filter(Boolean))];
      const view=expansionViews[0]||'';
      const operators=[...new Set(groupExpansions.map((item)=>String(item.operator||'AND').toUpperCase()))];
      rows.push({id:`${parameter.name}:${occurrence.ordinal}`,field:parameter.name,source:`${parameter.source?.kind||''}/${parameter.source?.name||''}`,type:parameter.typeExpr||'inferred',occurrence:occurrence.ordinal,predicate:normalizedPredicate,view,expansionViews,groupOperator:operators.length===1?operators[0]:operators.length?'MIXED':'UNEXPANDED',family:family(parameter.name),kind:String(predicate.name).toLowerCase()==='handler'?'handler':'sql'});
    }
  }
  return rows;
}

export function filterPredicateRows(rows,{query='',kind='all',family='all',view='all'}={}){
  const needle=query.trim().toLowerCase();
  return rows.filter((row)=>(kind==='all'||row.kind===kind)&&(family==='all'||row.family===family)&&(view==='all'||(row.expansionViews??[row.view]).includes(view))&&(!needle||[row.field,row.source,row.type,...(row.expansionViews??[row.view]),row.predicate?.name,...(row.predicate?.args??[])].join(' ').toLowerCase().includes(needle)));
}

export function predicatePage(rows,page=1,pageSize=25){const pages=Math.max(1,Math.ceil(rows.length/pageSize));const current=Math.min(Math.max(1,page),pages);return {items:rows.slice((current-1)*pageSize,current*pageSize),page:current,pages,total:rows.length};}

export function predicateGroups(structure,rows=predicateRows(structure)){
  const expansions=structure?.predicateExpansions??[];
  const byGroup=new Map();
  for(const row of rows){const items=byGroup.get(row.predicate.group)??[];items.push(row);byGroup.set(row.predicate.group,items);}
  const result=[];
  const expansionGroups=new Map();
  for(const site of expansions){const sites=expansionGroups.get(site.group)??[];sites.push(site);expansionGroups.set(site.group,sites);}
  for(const [group,sites] of expansionGroups){
    const views=[...new Set(sites.map((site)=>site.view).filter(Boolean))];
    const operators=[...new Set(sites.map((site)=>String(site.operator||'AND').toUpperCase()))];
    const methods=[...new Set(sites.map((site)=>site.method).filter(Boolean))];
    result.push({id:`group:${group}`,view:views[0]||'',group,operator:operators.length===1?operators[0]:'MIXED',methods,views,predicates:byGroup.get(group)??[],protected:group===99,ambiguous:operators.length>1});
  }
  for(const [group,predicates] of byGroup){
    if(expansions.some((site)=>site.group===group))continue;
    const view=predicates[0]?.view||'Unassigned';
    result.push({id:`${view}:${group}`.toLowerCase(),view,group,operator:'UNEXPANDED',views:[],predicates,protected:group===99,unexpanded:true});
  }
  return result.sort((a,b)=>a.view.localeCompare(b.view)||a.group-b.group);
}

export function filterPredicateGroups(groups,{query='',view='all'}={}){
  const needle=query.trim().toLowerCase();
  return groups.filter((group)=>(view==='all'||group.views.includes(view))&&(!needle||[...group.views,group.group,group.operator,...group.predicates.flatMap((row)=>[row.field,row.source,row.predicate?.name,...(row.predicate?.args??[])])].join(' ').toLowerCase().includes(needle)));
}

function family(name){if(/^Include/.test(name))return 'include';if(/^Exclude/.test(name))return 'exclude';return 'core';}
