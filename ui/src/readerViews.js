export function readerViews(structure) {
  const result=[];
  const add=(name,label=name)=>{if(name&&!result.some((item)=>item.name===name))result.push({name,label});};
  const visit=(view)=>{if(!view)return;const name=view.namespace||view.name;add(name,view.name&&view.name!==name?`${view.name} (${name})`:name);for(const relation of view.relations??[])visit(relation.view);};
  visit(structure?.component?.rootView);
  for(const view of structure?.views??[])add(view.name);
  return result;
}

// Column identities come from MCP inspection metadata, never from parsing SQL.
export function readerDimensions(structure, name) {
  let selected;
  const visit=(view)=>{if(!view)return;if((view.namespace||view.name)===name)selected=view;for(const relation of view.relations??[])visit(relation.view);};
  visit(structure?.component?.rootView);
  selected ||= (structure?.views??[]).find((view)=>(view.namespace||view.name)===name);
  return (selected?.columns??[]).map((column)=>({name:column.source||column.name,label:column.source&&column.source!==column.name?`${column.name} (${column.source})`:column.name}));
}
