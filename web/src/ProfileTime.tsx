import {useState} from 'react'
export type Policy={zone?:string;start?:string;end?:string;daily?:number;categories?:Record<string,number>}
export default function ProfileTime({value,change}:{value:Policy;change:(v:Policy)=>void}){
  const [name,setName]=useState('')
  const [minutes,setMinutes]=useState(15)
  const zone=value.zone||Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC'
  const update=(part:Policy)=>change({...value,zone,...part})
  return <details><summary>Viewing windows and daily allowances</summary>
    <p>Changes apply to new sessions. Daily minutes count from session start, including browsing and pauses. Category minutes count while a video in that category is selected.</p>
    <label>Household time zone<input maxLength={100} value={zone} onChange={e=>update({zone:e.target.value})}/></label>
    <label>Window opens<input type="time" required={Boolean(value.end)} value={value.start||''} onChange={e=>update({start:e.target.value})}/></label>
    <label>Window closes<input type="time" required={Boolean(value.start)} value={value.end||''} onChange={e=>update({end:e.target.value})}/></label>
    <label>Daily session minutes (0 for no limit)<input type="number" min="0" max="1440" value={value.daily||0} onChange={e=>update({daily:Number(e.target.value)})}/></label>
    <ul>{Object.entries(value.categories||{}).map(([category,limit])=><li key={category}>{category} · {limit} minutes <button type="button" aria-label={'Remove allowance for '+category} onClick={()=>{const categories={...value.categories};delete categories[category];update({categories})}}>Remove</button></li>)}</ul>
    <label>Custom category for allowance<input maxLength={400} value={name} onChange={e=>setName(e.target.value)}/></label>
    <label>Minutes per day in this category<input type="number" min="1" max="1440" value={minutes||''} onChange={e=>setMinutes(Number(e.target.value))}/></label>
    <button type="button" disabled={!name.trim()||!Number.isInteger(minutes)||minutes<1||minutes>1440} onClick={()=>{const key=name.trim().toLowerCase();update({categories:{...value.categories,[key]:minutes}});setName('')}}>Set category allowance</button>
  </details>
}
