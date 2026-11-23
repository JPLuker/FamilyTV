import {useState} from 'react'

type Program={title:string;start:number}
type GuideChannel={id:string;name:string;programs:Program[]}
function timestamp(value:string|null){
  const m=/^(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(?:\s*([+-])(\d{2})(\d{2}))?$/.exec(value||'')
  if(!m)throw new Error('A guide entry has an invalid start time.')
  const zone=m[7]?m[7]+m[8]+':'+m[9]:'Z'
  const t=Date.parse(m[1]+'-'+m[2]+'-'+m[3]+'T'+m[4]+':'+m[5]+':'+m[6]+zone)
  if(!Number.isFinite(t))throw new Error('A guide entry has an invalid start time.')
  return t
}
export function parseGuide(xml:string):GuideChannel[]{
  if(/<!DOCTYPE[^>]*\[/i.test(xml)||/<!ENTITY/i.test(xml))throw new Error('Guide contains unsupported XML declarations.')
  const doc=new DOMParser().parseFromString(xml,'application/xml')
  if(doc.querySelector('parsererror')||doc.documentElement.tagName!=='tv')throw new Error('Choose an XMLTV guide.')
  const channels=new Map<string,GuideChannel>()
  for(const element of doc.querySelectorAll('tv > channel')){
    const id=element.getAttribute('id'),name=element.querySelector('display-name')?.textContent?.trim()
    if(id&&name)channels.set(id,{id,name,programs:[]})
  }
  for(const el of doc.querySelectorAll('tv > programme')){
    const ch=channels.get(el.getAttribute('channel')||'')
    const title=el.querySelector('title')?.textContent?.trim()
    if(ch&&title)ch.programs.push({title,start:timestamp(el.getAttribute('start'))})
  }
  const populated=[...channels.values()].filter(c=>c.programs.length>0)
  if(!populated.length)throw new Error('No channels with programs were found.')
  for(const ch of populated)ch.programs.sort((a,b)=>a.start-b.start)
  return populated
}
export default function TunarrImport({media,apply}:{media:{id:string;title:string}[];apply:(name:string,lineup:string[])=>void}){
  const [channels,setChannels]=useState<GuideChannel[]>([])
  const [id,setID]=useState('')
  const [mapping,setMapping]=useState<Record<string,string>>({})
  const [error,setError]=useState('')
  const channel=channels.find(c=>c.id===id)
  const programs=channel?.programs.slice(0,200)||[]
  const titles=[...new Set(programs.map(p=>p.title))]
  function select(ch:GuideChannel){
    setID(ch.id)
    const pairs=ch.programs.slice(0,200).map(p=>{const matches=media.filter(m=>m.title.toLowerCase()===p.title.toLowerCase());return [p.title,matches.length===1?matches[0].id:'']})
    setMapping(Object.fromEntries(pairs))
  }
  async function read(file:File|undefined){
    if(!file)return
    setChannels([]);setID('');setError('')
    try{
      if(file.size>8*1024*1024)throw new Error('Choose an XML guide smaller than 8 MB.')
      const parsed=parseGuide(await file.text());setChannels(parsed);select(parsed[0])
    }catch(e){setError(e instanceof Error?e.message:'Could not read guide.')}
  }
  return <details><summary>Import a Tunarr guide</summary>
    <p>Save the XMLTV guide from your Tunarr server at /api/xmltv.xml. Choose a channel and match its programs to local videos. FamilyTV repeats the imported order from the time you create the channel.</p>
    <label>XMLTV file<input type="file" accept=".xml,application/xml,text/xml" onChange={e=>void read(e.target.files?.[0])}/></label>
    {error&&<p role="alert">{error}</p>}
    {channel&&<><label>Guide channel<select value={id} onChange={e=>{const ch=channels.find(c=>c.id===e.target.value);if(ch)select(ch)}}>{channels.map(c=><option key={c.id} value={c.id}>{c.name}</option>)}</select></label>
      <p>{programs.length} of {channel.programs.length} entries selected. Unmatched entries will be skipped.</p>
      {titles.map(title=><label key={title}>{title}<select value={mapping[title]||''} onChange={e=>setMapping({...mapping,[title]:e.target.value})}><option value="">Skip</option>{media.map(m=><option key={m.id} value={m.id}>{m.title}</option>)}</select></label>)}
      <button type="button" disabled={!programs.some(p=>mapping[p.title])} onClick={()=>apply(channel.name,programs.map(p=>mapping[p.title]).filter(Boolean))}>Use matching programs</button></>}
  </details>
}
