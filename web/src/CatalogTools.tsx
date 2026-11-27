import {useEffect,useRef,useState} from 'react'
import {errorMessage,request} from './api'
import type {ReviewedMedia} from './MediaReview'
import LibraryFolder from './LibraryFolder'
import Lookup from './Lookup'
export type Metadata={playlist_index?:number;attribution?:string;title:string;kind:string;series:string;season:number;episode:number;creator:string;playlist:string;external_id:string;language:string;license:string;source:string;collection:string}
type ScanStatus={running:boolean;finished:number;imported:number;skipped:number;failed:number;last_error?:string}
export function Scanner({refresh}:{refresh:()=>void}){
  const [scan,setScan]=useState<ScanStatus>()
  const [submitting,setSubmitting]=useState(false)
  const [error,setError]=useState('')
  const [revision,setRevision]=useState(0)
  const lastFinish=useRef(0)
  useEffect(()=>{
    let live=true
    let timer:ReturnType<typeof setTimeout>
    async function poll(){
      let delay=10000
      try{
        const state=await request<ScanStatus>('media/library-scan')
        if(!live)return
        setError('');setScan(state)
        if(state.finished>lastFinish.current){lastFinish.current=state.finished;refresh()}
        if(state.running)delay=800
      }catch(e){if(live)setError(errorMessage(e))}
      if(live)timer=setTimeout(poll,delay)
    }
    void poll()
    return()=>{live=false;clearTimeout(timer)}
  },[refresh,revision])
  async function start(){
    setSubmitting(true)
    try{setScan(await request<ScanStatus>('media/library-scan','POST'));setRevision(n=>n+1)}
    catch(e){setError(errorMessage(e))}finally{setSubmitting(false)}
  }
  return <section><h3>Library scan</h3>
    <LibraryFolder scanning={Boolean(scan?.running)} onScan={()=>setRevision(n=>n+1)}/>
    <p>At startup and every hour FamilyTV imports new videos from your media folder and its subfolders. Review them together in the approval queue.</p>
    <button id="scan-library" disabled={submitting||scan?.running} onClick={start}>Scan now</button>
    {scan&&(scan.running||scan.finished>0)&&<p role="status">{scan.running?'Scanning…':'Scan complete.'} Imported {scan.imported} · skipped {scan.skipped} · failed {scan.failed}. {scan.last_error} {!scan.running&&!scan.imported&&!scan.skipped&&!scan.failed&&'No supported videos found. Choose the folder that contains your videos.'}</p>}
    {error&&<p role="alert">{error}</p>}
  </section>
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
    <p>Duration · {item.duration_seconds.toFixed(1)} s</p>
    {item.streams?.length ? <ul aria-label="Media tracks">{item.streams.map((track,i)=><li key={i}>{track.codec_type} · {track.codec_name||'Unknown codec'}</li>)}</ul> :
      <button type="button" disabled={busy} onClick={async()=>{setBusy(true);try{saved(await request('media/'+item.id+'/probe','POST'))}catch(e){setError(errorMessage(e))}finally{setBusy(false)}}}>Read codecs from file</button>}
    <label>Playlist position<input type="number" min="1" value={value.playlist_index||''} onChange={e=>setValue({...value,playlist_index:Number(e.target.value)})}/></label>
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
