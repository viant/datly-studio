import {readerViews} from './readerViews.js';
export const singleArgumentPredicate=(name)=>['expr','literal_in','multi_in','multi_not_in','when_present','when_not_present','handler'].includes(name);
export function predicateOptions(structure,current='') {const names=structure?.availablePredicates??[];return [...new Set([...names,current].filter(Boolean))];}
export function newPredicateDraft(structure,name='equal') {return {name,args:'',view:readerViews(structure)[0]?.name||'',group:1,groupOperator:'AND',applyWhenAbsent:false};}
export function predicateArguments(draft) {const value=String(draft.args||'').trim();if(!value)return [];return singleArgumentPredicate(draft.name)?[value]:value.split(value.includes('\n')?/\r?\n/:',').map((item)=>item.trim()).filter(Boolean);}
export function predicateMutation(draft,field,occurrence=0) {
 const args=predicateArguments(draft);
 if(!field||!draft.name||!draft.view||!Number.isInteger(draft.group)||draft.group<0||!args.length)throw new Error('Choose a predicate, target view, arguments, and a non-negative group.');
 return {field,occurrence,view:draft.view,expansionViews:draft.expansionViews?.length?draft.expansionViews:undefined,group:draft.group,name:draft.name,args,applyWhenAbsent:draft.applyWhenAbsent===true,groupOperator:draft.groupOperator};
}
