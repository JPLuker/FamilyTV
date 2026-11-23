import {useState} from 'react'
import {errorMessage,request} from './api'
import type {ReviewedMedia} from './MediaReview'
import Lookup from './Lookup'
export type Metadata={attribution?:string;title:string;kind:string;series:string;season:number;episode:number;creator:string;playlist:string;external_id:string;language:string;license:string;source:string;collection:string}
export function Scanner({media,added}:{media:ReviewedMedia[];added:(m:ReviewedMedia)=>void}){
  const [paths,setPaths]=useState<string[]>([])
  const [selected,setSelected]=useState<string[]>([])
  const [more,setMore]=useState(false)
  const [busy,setBusy]=useState(false)
  const [status,setStatus]=useState('')
  async function scan(append:boolean){
    setBusy(true)
    try{const result=await request<{paths:string[];more:boolean;skipped:number}>('media/scan'+(append?'?after='+encodeURIComponent(paths.at(-1)||''):''))
      setPaths(append?[...paths,...result.paths]:result.paths);setMore(result.more);setSelected([]);setStatus(result.paths.length+' files on this page. '+result.skipped+' unreadable paths.')
    }catch(e){setStatus(errorMessage(e))}finally{setBusy(false)}
  }
  async function importSelected(){
    setBusy(true);let count=0;const failed:string[]=[]
    for(const path of selected){try{added(await request<ReviewedMedia>('media','POST',{path}));count++}catch(e){failed.push(path+' — '+errorMessage(e))}}
    setStatus('Imported '+count+'. '+failed.join(' · '));setSelected([]);setBusy(false)
  }
  return <details><summary>Scan the media folder</summary>
    <button disabled={busy} onClick={()=>scan(false)}>Scan files</button>
    {paths.map(path=><label key={path} className="toggle"><input type="checkbox" disabled={busy||media.some(m=>m.path===path)} checked={selected.includes(path)} onChange={e=>setSelected(old=>e.target.checked?[...old,path]:old.filter(p=>p!==path))}/>{path}{media.some(m=>m.path===path)?' · imported':''}</label>)}
    {more&&<button disabled={busy} onClick={()=>scan(true)}>Load more files</button>}
    {paths.length>0&&<button disabled={busy||!selected.length} onClick={importSelected}>Import selected files</button>}
    {status&&<p role="status">{status}</p>}
  </details>
}
export function MetadataEditor({item,saved,close}:{item:ReviewedMedia;saved:(m:ReviewedMedia)=>void;close:()=>void}){
  const [value,setValue]=useState<Metadata>({...item.metadata,title:item.title,kind:item.metadata.kind||'video'})
  const [error,setError]=useState('')
  const [busy,setBusy]=useState(false)
  async function update(reset:boolean){
    setBusy(true);setError('')
    try{saved(await request('media/'+item.id+'/metadata',reset?'DELETE':'PUT',reset?undefined:value))}
    catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  return <form onSubmit={e=>{e.preventDefault();void update(false)}}><h3>Edit catalog details</h3>
    <Lookup value={value} change={setValue}/>
    {value.attribution&&<p>Provider attribution · <a href={value.attribution} rel="noreferrer" target="_blank">Original listing</a></p>}
    <p>Metadata source · {item.metadata.source||'Imported filename'}</p>
    <label>Display title<input required maxLength={160} value={value.title} onChange={e=>setValue({...value,title:e.target.value})}/></label>
    <label>Format<select value={value.kind} onChange={e=>setValue({...value,kind:e.target.value})}>{['video','movie','episode','special','creator'].map(kind=><option key={kind}>{kind}</option>)}</select></label>
    {([['series','Series'],['creator','Creator'],['playlist','Playlist'],['collection','Collection'],['language','Language'],['external_id','External video ID'],['license','Source license']] as const).map(([key,label])=><label key={key}>{label}<input maxLength={250} value={value[key]} onChange={e=>setValue({...value,[key]:e.target.value})}/></label>)}
    {(['season','episode'] as const).map(key=><label key={key}>{key==='season'?'Season number':'Episode number'}<input type="number" min="0" value={value[key]} onChange={e=>setValue({...value,[key]:Number(e.target.value)})}/></label>)}
    {error&&<p role="alert">{error}</p>}<button disabled={busy}>Save catalog details</button>
    <button type="button" disabled={busy} onClick={()=>update(true)}>Restore imported metadata</button><button type="button" onClick={close}>Cancel</button>
  </form>
}
