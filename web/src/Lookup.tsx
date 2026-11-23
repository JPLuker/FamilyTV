import {useState} from 'react'
import {errorMessage,request} from './api'
import type {Metadata} from './CatalogTools'
type Match={title:string;provider:string;external_id:string;url:string;creator?:string}
export default function Lookup({value,change}:{value:Metadata;change:(v:Metadata)=>void}){
  const [provider,setProvider]=useState('tvmaze'),[term,setTerm]=useState(''),[matches,setMatches]=useState<Match[]>([])
  const [status,setStatus]=useState(''),[busy,setBusy]=useState(false)
  async function search(){
    setBusy(true);setMatches([]);setStatus('')
    try{const {matches}=await request<{matches:Match[]}>('metadata/lookup?'+new URLSearchParams({provider,q:term}));setMatches(matches);setStatus(matches.length+' matches.')}
    catch(e){setStatus(errorMessage(e))}finally{setBusy(false)}
  }
  return <details><summary>Look up optional metadata</summary>
    <label>Metadata provider<select value={provider} onChange={e=>{setProvider(e.target.value);setMatches([]);setTerm('')}}><option value="tvmaze">TVmaze series</option><option value="youtube">YouTube video</option></select></label>
    <label>{provider==='youtube'?'YouTube video ID':'Series title'}<input maxLength={180} value={term} onChange={e=>setTerm(e.target.value)}/></label>
    <button type="button" disabled={busy||!term.trim()} onClick={search}>Send lookup to {provider==='youtube'?'YouTube':'TVmaze'}</button>
    <p role="status">{status}</p><ul>{matches.map(m=><li key={m.external_id}>{m.title} <button type="button" onClick={()=>change(provider==='youtube'?{...value,title:m.title,creator:m.creator||'',kind:'creator',external_id:m.external_id,attribution:m.url}:{...value,series:m.title,external_id:m.external_id,attribution:m.url})}>Use match</button></li>)}</ul>
  </details>
}
