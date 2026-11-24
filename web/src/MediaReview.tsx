import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'
import type {Metadata} from './CatalogTools'
export type Review={age_band?:string;category?:string;goal?:string;reviewer?:string;reference?:string;confidence?:string;together?:string;activity?:string;profiles:string[]|null;subject:string;audio_description:string;caption_quality:string}
export type ReviewedMedia={review_version?:number;reviewed?:boolean;streams?:{codec_type:string;codec_name?:string}[];metadata:Metadata;id:string;title:string;path:string;duration_seconds:number;captions:boolean;review:Review}
export default function MediaReview({item,saved,close}:{item:ReviewedMedia;saved:(m:ReviewedMedia)=>void;close:()=>void}){
  const [profiles,setProfiles]=useState<{id:string;name:string}[]>([])
  const [error,setError]=useState('')
  const [ready,setReady]=useState(false)
  const [busy,setBusy]=useState(false)
  const [review,setReview]=useState<Review>({...item.review,profiles:item.review.profiles||[],audio_description:item.review.audio_description||'unknown',caption_quality:item.review.caption_quality||'unknown'})
  useEffect(()=>{let active=true;request<{profiles:{id:string;name:string}[]}>('profiles').then(result=>{
    if(active){
      setProfiles(result.profiles)
      setReview(old=>({...old,profiles:old.profiles?.filter(id=>result.profiles.some(p=>p.id===id))||[]}))
      setReady(true)
    }
  }).catch(e=>{if(active)setError(errorMessage(e))});return()=>{active=false}},[])
  async function submit(e:React.FormEvent){
    e.preventDefault();setBusy(true);setError('')
    try{saved(await request<ReviewedMedia>('media/'+item.id+'/review','PUT',{...review,expected_version:item.review_version||0}))}catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  return <form onSubmit={submit}><h3 id="media-review-heading" tabIndex={-1}>Review {item.title}</h3>
    <label>Subject<input maxLength={100} value={review.subject} onChange={e=>setReview({...review,subject:e.target.value})}/></label>
    {([['caption_quality','Caption quality'],['audio_description','Audio description in the default soundtrack']] as const).map(([key,label])=><label key={key}>{label}
      <select value={review[key]} onChange={e=>setReview({...review,[key]:e.target.value})}>
        <option value="unknown">Unknown</option><option value="verified" disabled={key==='caption_quality'&&!item.captions}>Verified by parent</option><option value="missing">Missing</option>
      </select></label>)}
    <details><summary>Education and family notes</summary>
      <label>Suggested age band<select value={review.age_band||''} onChange={e=>setReview({...review,age_band:e.target.value})}><option value="">Unspecified</option>{['2-4','5-7','8-11','12-14','15-17'].map(b=><option key={b}>{b}</option>)}</select></label>
      {([['category','Custom category'],['goal','Claimed learning goal'],['reviewer','Reviewer'],['reference','Source or trusted review reference'],['together','Co-viewing tip'],['activity','Off-screen activity']] as const).map(([key,label])=><label key={key}>{label}<input maxLength={400} value={review[key]||''} onChange={e=>setReview({...review,[key]:e.target.value})}/></label>)}
      <label>Learning goal evidence<select value={review.confidence||''} onChange={e=>setReview({...review,confidence:e.target.value})}><option value="">Unknown</option><option value="described">Content description</option><option value="source-reviewed">Source checked by reviewer</option></select></label>
    </details>
    <fieldset><legend>Approved for</legend>{profiles.map(p=><label className="toggle" key={p.id}>
      <input type="checkbox" checked={review.profiles?.includes(p.id)||false} onChange={e=>setReview({...review,profiles:e.target.checked?[...(review.profiles||[]),p.id]:(review.profiles||[]).filter(id=>id!==p.id)})}/>{p.name}</label>)}
      {ready&&!profiles.length&&<p>Create a child profile to approve this video.</p>}</fieldset>
    {error&&<p role="alert">{error}</p>}
    <button disabled={busy||!ready}>Save review</button><button type="button" disabled={busy} onClick={close}>Cancel review</button></form>
}
