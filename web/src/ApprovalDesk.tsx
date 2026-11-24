import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'
import type {ReviewedMedia} from './MediaReview'

type Profile={id:string;name:string}
type Event={titles:string[];profile_ids:string[]|null;action:string;at:number;target:string;count:number}
type Coverage={approved:number;playable:number;short_captioned:number;counts:Record<string,Record<string,number>>;excluded:Record<string,number>}

export default function ApprovalDesk({media,refresh,review}:{media:ReviewedMedia[];refresh:()=>void;review:(m:ReviewedMedia)=>void}){
  const [open,setOpen]=useState(false)
  const [profiles,setProfiles]=useState<Profile[]>([])
  const [profile,setProfile]=useState('')
  const [events,setEvents]=useState<Event[]>([])
  const [coverage,setCoverage]=useState<Coverage>()
  const [filter,setFilter]=useState('pending')
  const [query,setQuery]=useState('')
  const [group,setGroup]=useState('')
  const groups=Array.from(new Set(media.flatMap(m=>[m.metadata.series?'Series · '+m.metadata.series:'',m.metadata.creator?'Creator · '+m.metadata.creator:'']).filter(Boolean))).sort()
  const [chosen,setChosen]=useState<string[]>([])
  const [error,setError]=useState('')
  const [notice,setNotice]=useState('')
  const [busy,setBusy]=useState(false)
  const [revision,setRevision]=useState(0)
  const [tips,setTips]=useState(()=>localStorage.getItem('familytv-coverage-tips')!=='off')
  const rows=media.filter(m=>(!group||group==='Series · '+m.metadata.series||group==='Creator · '+m.metadata.creator)&&(filter==='all'||(!m.reviewed&&!m.review.profiles?.length))&&[m.title,m.metadata.series,m.metadata.creator,m.review.category].some(value=>value?.toLowerCase().includes(query.toLowerCase())))
  useEffect(()=>{
    if(!open)return
    let current=true
    request<{events:Event[]|null;profiles:Profile[]|null}>('approvals').then(result=>{
      if(!current)return
      const next=result.profiles||[]
      setProfiles(next);setEvents(result.events||[])
      setProfile(id=>next.some(p=>p.id===id)?id:next[0]?.id||'')
    }).catch(e=>{if(current)setError(errorMessage(e))})
    return ()=>{current=false}
  },[open,media,revision])
  useEffect(()=>{
    setCoverage(undefined)
    if(!open||!profile)return
    let current=true
    request<Coverage>('coverage/'+encodeURIComponent(profile)).then(c=>{if(current)setCoverage(c)}).catch(e=>{if(current)setError(errorMessage(e))})
    return ()=>{current=false}
  },[open,profile,media,revision])
  async function apply(action:'approve'|'revoke'){
    setBusy(true);setError('');setNotice('')
    try{
      const result=await request<{changed:number}>('approvals','POST',{ids:chosen,profile,action})
      setNotice('Updated '+result.changed+' videos.');setChosen([]);refresh()
    }catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  function toggleTips(){localStorage.setItem('familytv-coverage-tips',tips?'off':'on');setTips(!tips)}
  return <details onToggle={e=>setOpen(e.currentTarget.open)}><summary>Approval queue and library coverage</summary>
    {open&&<>
      <label>Child profile for approvals and coverage<select disabled={busy} value={profile} onChange={e=>{setChosen([]);setProfile(e.target.value)}}>{profiles.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}</select></label>
      <label>Queue view<select disabled={busy} value={filter} onChange={e=>{setChosen([]);setFilter(e.target.value)}}><option value="pending">Awaiting review</option><option value="all">All videos</option></select></label>
      {groups.length>0&&<label>Series or creator<select disabled={busy} value={group} onChange={e=>{setChosen([]);setGroup(e.target.value)}}><option value="">All series and creators</option>{groups.map(name=><option key={name}>{name}</option>)}</select></label>}
      <label>Filter by title, series, creator or category<input disabled={busy} value={query} onChange={e=>{setChosen([]);setQuery(e.target.value)}}/></label>
      {rows.length===0?<p>No matching videos.</p>:<fieldset disabled={busy}><legend>Videos to update</legend>
        <button type="button" onClick={()=>setChosen(rows.slice(0,100).map(m=>m.id))}>Select {Math.min(rows.length,100)} matching videos</button>
        {rows.map(m=><div key={m.id}><label className="toggle"><input type="checkbox" checked={chosen.includes(m.id)} disabled={!chosen.includes(m.id)&&chosen.length>=100} onChange={e=>setChosen(e.target.checked?[...chosen,m.id]:chosen.filter(id=>id!==m.id))}/>{m.title}</label>
          <button type="button" onClick={()=>review(m)}>Review details for {m.title}</button></div>)}
      </fieldset>}
      <div className="actions">{(['approve','revoke'] as const).map(action=><button key={action} disabled={busy||!chosen.length||!profile} onClick={()=>apply(action)}>{action==='approve'?'Approve':'Revoke'} {chosen.length} selected</button>)}</div>
      {notice&&<p role="status">{notice}</p>}
      {error&&<p role="alert">{error} <button onClick={()=>{setError('');setRevision(n=>n+1)}}>Retry</button></p>}
      {coverage&&<section aria-label="Coverage for selected profile"><h3>Library coverage</h3>
        <p>{coverage.approved} approved · {coverage.playable} playable before session limits</p>
        {Object.entries(coverage.excluded).map(([reason,n])=><p key={reason}>{reason}: {n}</p>)}
        {Object.entries(coverage.counts).map(([dimension,counts])=><details key={dimension}><summary>{dimension}</summary><ul>{Object.entries(counts).map(([name,n])=><li key={name}>{name} · {n}</li>)}</ul></details>)}
        {tips&&coverage.short_captioned===0&&<p>No playable videos under ten minutes have verified captions.</p>}
        <button onClick={toggleTips}>{tips?'Hide coverage suggestions':'Show coverage suggestions'}</button>
      </section>}
      <details><summary>Parent review history</summary>{events.length?<ol>{events.slice().reverse().map((entry,i)=><li key={i}>{new Date(entry.at).toLocaleString()} · {entry.action} · {entry.target} · {(entry.profile_ids||[]).map(id=>profiles.find(p=>p.id===id)?.name||"Removed profile").join(", ")||"No profiles"} · {entry.titles.join(", ")}</li>)}</ol>:<p>No review actions recorded yet.</p>}</details>
    </>}
  </details>
}
