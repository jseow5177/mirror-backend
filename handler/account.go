package handler

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"math/rand"
	"strings"
	"time"
)

const (
	targetLink            = "https://www.mirrorcdp.com/"
	dummyEmailHtmlEncoded = "PCFET0NUWVBFIEhUTUwgUFVCTElDICItLy9XM0MvL0RURCBYSFRNTCAxLjAgVHJhbnNpdGlvbmFsIC8vRU4iICJodHRwOi8vd3d3LnczLm9yZy9UUi94aHRtbDEvRFREL3hodG1sMS10cmFuc2l0aW9uYWwuZHRkIj4KPGh0bWwgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGh0bWwiIHhtbG5zOnY9InVybjpzY2hlbWFzLW1pY3Jvc29mdC1jb206dm1sIiB4bWxuczpvPSJ1cm46c2NoZW1hcy1taWNyb3NvZnQtY29tOm9mZmljZTpvZmZpY2UiPgo8aGVhZD4KPCEtLVtpZiBndGUgbXNvIDldPgo8eG1sPgogIDxvOk9mZmljZURvY3VtZW50U2V0dGluZ3M+CiAgICA8bzpBbGxvd1BORy8+CiAgICA8bzpQaXhlbHNQZXJJbmNoPjk2PC9vOlBpeGVsc1BlckluY2g+CiAgPC9vOk9mZmljZURvY3VtZW50U2V0dGluZ3M+CjwveG1sPgo8IVtlbmRpZl0tLT4KICA8bWV0YSBodHRwLWVxdWl2PSJDb250ZW50LVR5cGUiIGNvbnRlbnQ9InRleHQvaHRtbDsgY2hhcnNldD1VVEYtOCI+CiAgPG1ldGEgbmFtZT0idmlld3BvcnQiIGNvbnRlbnQ9IndpZHRoPWRldmljZS13aWR0aCwgaW5pdGlhbC1zY2FsZT0xLjAiPgogIDxtZXRhIG5hbWU9IngtYXBwbGUtZGlzYWJsZS1tZXNzYWdlLXJlZm9ybWF0dGluZyI+CiAgPCEtLVtpZiAhbXNvXT48IS0tPjxtZXRhIGh0dHAtZXF1aXY9IlgtVUEtQ29tcGF0aWJsZSIgY29udGVudD0iSUU9ZWRnZSI+PCEtLTwhW2VuZGlmXS0tPgogIDx0aXRsZT48L3RpdGxlPgogIAogICAgPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KICAgICAgCiAgICAgIEBtZWRpYSBvbmx5IHNjcmVlbiBhbmQgKG1pbi13aWR0aDogNTIwcHgpIHsKICAgICAgICAudS1yb3cgewogICAgICAgICAgd2lkdGg6IDUwMHB4ICFpbXBvcnRhbnQ7CiAgICAgICAgfQoKICAgICAgICAudS1yb3cgLnUtY29sIHsKICAgICAgICAgIHZlcnRpY2FsLWFsaWduOiB0b3A7CiAgICAgICAgfQoKICAgICAgICAKICAgICAgICAgICAgLnUtcm93IC51LWNvbC0xMDAgewogICAgICAgICAgICAgIHdpZHRoOiA1MDBweCAhaW1wb3J0YW50OwogICAgICAgICAgICB9CiAgICAgICAgICAKICAgICAgfQoKICAgICAgQG1lZGlhIG9ubHkgc2NyZWVuIGFuZCAobWF4LXdpZHRoOiA1MjBweCkgewogICAgICAgIC51LXJvdy1jb250YWluZXIgewogICAgICAgICAgbWF4LXdpZHRoOiAxMDAlICFpbXBvcnRhbnQ7CiAgICAgICAgICBwYWRkaW5nLWxlZnQ6IDBweCAhaW1wb3J0YW50OwogICAgICAgICAgcGFkZGluZy1yaWdodDogMHB4ICFpbXBvcnRhbnQ7CiAgICAgICAgfQoKICAgICAgICAudS1yb3cgewogICAgICAgICAgd2lkdGg6IDEwMCUgIWltcG9ydGFudDsKICAgICAgICB9CgogICAgICAgIC51LXJvdyAudS1jb2wgewogICAgICAgICAgZGlzcGxheTogYmxvY2sgIWltcG9ydGFudDsKICAgICAgICAgIHdpZHRoOiAxMDAlICFpbXBvcnRhbnQ7CiAgICAgICAgICBtaW4td2lkdGg6IDMyMHB4ICFpbXBvcnRhbnQ7CiAgICAgICAgICBtYXgtd2lkdGg6IDEwMCUgIWltcG9ydGFudDsKICAgICAgICB9CgogICAgICAgIC51LXJvdyAudS1jb2wgPiBkaXYgewogICAgICAgICAgbWFyZ2luOiAwIGF1dG87CiAgICAgICAgfQoKCn0KICAgIApib2R5e21hcmdpbjowO3BhZGRpbmc6MH10YWJsZSx0ZCx0cntib3JkZXItY29sbGFwc2U6Y29sbGFwc2U7dmVydGljYWwtYWxpZ246dG9wfXB7bWFyZ2luOjB9LmllLWNvbnRhaW5lciB0YWJsZSwubXNvLWNvbnRhaW5lciB0YWJsZXt0YWJsZS1sYXlvdXQ6Zml4ZWR9KntsaW5lLWhlaWdodDppbmhlcml0fWFbeC1hcHBsZS1kYXRhLWRldGVjdG9ycz10cnVlXXtjb2xvcjppbmhlcml0IWltcG9ydGFudDt0ZXh0LWRlY29yYXRpb246bm9uZSFpbXBvcnRhbnR9CgoKdGFibGUsIHRkIHsgY29sb3I6ICMwMDAwMDA7IH0gI3VfYm9keSBhIHsgY29sb3I6ICMwMDAwZWU7IHRleHQtZGVjb3JhdGlvbjogdW5kZXJsaW5lOyB9CiAgICA8L3N0eWxlPgogIAogIAoKPC9oZWFkPgoKPGJvZHkgY2xhc3M9ImNsZWFuLWJvZHkgdV9ib2R5IiBzdHlsZT0ibWFyZ2luOiAwO3BhZGRpbmc6IDA7LXdlYmtpdC10ZXh0LXNpemUtYWRqdXN0OiAxMDAlO2JhY2tncm91bmQtY29sb3I6ICNGN0Y4Rjk7Y29sb3I6ICMwMDAwMDAiPgogIDwhLS1baWYgSUVdPjxkaXYgY2xhc3M9ImllLWNvbnRhaW5lciI+PCFbZW5kaWZdLS0+CiAgPCEtLVtpZiBtc29dPjxkaXYgY2xhc3M9Im1zby1jb250YWluZXIiPjwhW2VuZGlmXS0tPgogIDx0YWJsZSBpZD0idV9ib2R5IiBzdHlsZT0iYm9yZGVyLWNvbGxhcHNlOiBjb2xsYXBzZTt0YWJsZS1sYXlvdXQ6IGZpeGVkO2JvcmRlci1zcGFjaW5nOiAwO21zby10YWJsZS1sc3BhY2U6IDBwdDttc28tdGFibGUtcnNwYWNlOiAwcHQ7dmVydGljYWwtYWxpZ246IHRvcDttaW4td2lkdGg6IDMyMHB4O01hcmdpbjogMCBhdXRvO2JhY2tncm91bmQtY29sb3I6ICNGN0Y4Rjk7d2lkdGg6MTAwJSIgY2VsbHBhZGRpbmc9IjAiIGNlbGxzcGFjaW5nPSIwIj4KICA8dGJvZHk+CiAgPHRyIHN0eWxlPSJ2ZXJ0aWNhbC1hbGlnbjogdG9wIj4KICAgIDx0ZCBzdHlsZT0id29yZC1icmVhazogYnJlYWstd29yZDtib3JkZXItY29sbGFwc2U6IGNvbGxhcHNlICFpbXBvcnRhbnQ7dmVydGljYWwtYWxpZ246IHRvcCI+CiAgICA8IS0tW2lmIChtc28pfChJRSldPjx0YWJsZSB3aWR0aD0iMTAwJSIgY2VsbHBhZGRpbmc9IjAiIGNlbGxzcGFjaW5nPSIwIiBib3JkZXI9IjAiPjx0cj48dGQgYWxpZ249ImNlbnRlciIgc3R5bGU9ImJhY2tncm91bmQtY29sb3I6ICNGN0Y4Rjk7Ij48IVtlbmRpZl0tLT4KICAgIAogIAogIAo8ZGl2IGNsYXNzPSJ1LXJvdy1jb250YWluZXIiIHN0eWxlPSJwYWRkaW5nOiAwcHg7YmFja2dyb3VuZC1jb2xvcjogdHJhbnNwYXJlbnQiPgogIDxkaXYgY2xhc3M9InUtcm93IiBzdHlsZT0ibWFyZ2luOiAwIGF1dG87bWluLXdpZHRoOiAzMjBweDttYXgtd2lkdGg6IDUwMHB4O292ZXJmbG93LXdyYXA6IGJyZWFrLXdvcmQ7d29yZC13cmFwOiBicmVhay13b3JkO3dvcmQtYnJlYWs6IGJyZWFrLXdvcmQ7YmFja2dyb3VuZC1jb2xvcjogdHJhbnNwYXJlbnQ7Ij4KICAgIDxkaXYgc3R5bGU9ImJvcmRlci1jb2xsYXBzZTogY29sbGFwc2U7ZGlzcGxheTogdGFibGU7d2lkdGg6IDEwMCU7aGVpZ2h0OiAxMDAlO2JhY2tncm91bmQtY29sb3I6IHRyYW5zcGFyZW50OyI+CiAgICAgIDwhLS1baWYgKG1zbyl8KElFKV0+PHRhYmxlIHdpZHRoPSIxMDAlIiBjZWxscGFkZGluZz0iMCIgY2VsbHNwYWNpbmc9IjAiIGJvcmRlcj0iMCI+PHRyPjx0ZCBzdHlsZT0icGFkZGluZzogMHB4O2JhY2tncm91bmQtY29sb3I6IHRyYW5zcGFyZW50OyIgYWxpZ249ImNlbnRlciI+PHRhYmxlIGNlbGxwYWRkaW5nPSIwIiBjZWxsc3BhY2luZz0iMCIgYm9yZGVyPSIwIiBzdHlsZT0id2lkdGg6NTAwcHg7Ij48dHIgc3R5bGU9ImJhY2tncm91bmQtY29sb3I6IHRyYW5zcGFyZW50OyI+PCFbZW5kaWZdLS0+CiAgICAgIAo8IS0tW2lmIChtc28pfChJRSldPjx0ZCBhbGlnbj0iY2VudGVyIiB3aWR0aD0iNTAwIiBzdHlsZT0id2lkdGg6IDUwMHB4O3BhZGRpbmc6IDBweDtib3JkZXItdG9wOiAwcHggc29saWQgdHJhbnNwYXJlbnQ7Ym9yZGVyLWxlZnQ6IDBweCBzb2xpZCB0cmFuc3BhcmVudDtib3JkZXItcmlnaHQ6IDBweCBzb2xpZCB0cmFuc3BhcmVudDtib3JkZXItYm90dG9tOiAwcHggc29saWQgdHJhbnNwYXJlbnQ7Ym9yZGVyLXJhZGl1czogMHB4Oy13ZWJraXQtYm9yZGVyLXJhZGl1czogMHB4OyAtbW96LWJvcmRlci1yYWRpdXM6IDBweDsiIHZhbGlnbj0idG9wIj48IVtlbmRpZl0tLT4KPGRpdiBjbGFzcz0idS1jb2wgdS1jb2wtMTAwIiBzdHlsZT0ibWF4LXdpZHRoOiAzMjBweDttaW4td2lkdGg6IDUwMHB4O2Rpc3BsYXk6IHRhYmxlLWNlbGw7dmVydGljYWwtYWxpZ246IHRvcDsiPgogIDxkaXYgc3R5bGU9ImhlaWdodDogMTAwJTt3aWR0aDogMTAwJSAhaW1wb3J0YW50O2JvcmRlci1yYWRpdXM6IDBweDstd2Via2l0LWJvcmRlci1yYWRpdXM6IDBweDsgLW1vei1ib3JkZXItcmFkaXVzOiAwcHg7Ij4KICA8IS0tW2lmICghbXNvKSYoIUlFKV0+PCEtLT48ZGl2IHN0eWxlPSJib3gtc2l6aW5nOiBib3JkZXItYm94OyBoZWlnaHQ6IDEwMCU7IHBhZGRpbmc6IDBweDtib3JkZXItdG9wOiAwcHggc29saWQgdHJhbnNwYXJlbnQ7Ym9yZGVyLWxlZnQ6IDBweCBzb2xpZCB0cmFuc3BhcmVudDtib3JkZXItcmlnaHQ6IDBweCBzb2xpZCB0cmFuc3BhcmVudDtib3JkZXItYm90dG9tOiAwcHggc29saWQgdHJhbnNwYXJlbnQ7Ym9yZGVyLXJhZGl1czogMHB4Oy13ZWJraXQtYm9yZGVyLXJhZGl1czogMHB4OyAtbW96LWJvcmRlci1yYWRpdXM6IDBweDsiPjwhLS08IVtlbmRpZl0tLT4KICAKPHRhYmxlIHN0eWxlPSJmb250LWZhbWlseTphcmlhbCxoZWx2ZXRpY2Esc2Fucy1zZXJpZjsiIHJvbGU9InByZXNlbnRhdGlvbiIgY2VsbHBhZGRpbmc9IjAiIGNlbGxzcGFjaW5nPSIwIiB3aWR0aD0iMTAwJSIgYm9yZGVyPSIwIj4KICA8dGJvZHk+CiAgICA8dHI+CiAgICAgIDx0ZCBzdHlsZT0ib3ZlcmZsb3ctd3JhcDpicmVhay13b3JkO3dvcmQtYnJlYWs6YnJlYWstd29yZDtwYWRkaW5nOjEwcHg7Zm9udC1mYW1pbHk6YXJpYWwsaGVsdmV0aWNhLHNhbnMtc2VyaWY7IiBhbGlnbj0ibGVmdCI+CiAgICAgICAgCiAgPCEtLVtpZiBtc29dPjx0YWJsZSB3aWR0aD0iMTAwJSI+PHRyPjx0ZD48IVtlbmRpZl0tLT4KICAgIDxoMSBzdHlsZT0ibWFyZ2luOiAwcHg7IGxpbmUtaGVpZ2h0OiAxNDAlOyB0ZXh0LWFsaWduOiBjZW50ZXI7IHdvcmQtd3JhcDogYnJlYWstd29yZDsgZm9udC1zaXplOiAyMnB4OyBmb250LXdlaWdodDogNDAwOyI+PHNwYW4+V2VsY29tZSB0byBNaXJyb3IhPC9zcGFuPjwvaDE+CiAgPCEtLVtpZiBtc29dPjwvdGQ+PC90cj48L3RhYmxlPjwhW2VuZGlmXS0tPgoKICAgICAgPC90ZD4KICAgIDwvdHI+CiAgPC90Ym9keT4KPC90YWJsZT4KCjx0YWJsZSBzdHlsZT0iZm9udC1mYW1pbHk6YXJpYWwsaGVsdmV0aWNhLHNhbnMtc2VyaWY7IiByb2xlPSJwcmVzZW50YXRpb24iIGNlbGxwYWRkaW5nPSIwIiBjZWxsc3BhY2luZz0iMCIgd2lkdGg9IjEwMCUiIGJvcmRlcj0iMCI+CiAgPHRib2R5PgogICAgPHRyPgogICAgICA8dGQgc3R5bGU9Im92ZXJmbG93LXdyYXA6YnJlYWstd29yZDt3b3JkLWJyZWFrOmJyZWFrLXdvcmQ7cGFkZGluZzoxMHB4O2ZvbnQtZmFtaWx5OmFyaWFsLGhlbHZldGljYSxzYW5zLXNlcmlmOyIgYWxpZ249ImxlZnQiPgogICAgICAgIAogIDx0YWJsZSBoZWlnaHQ9IjBweCIgYWxpZ249ImNlbnRlciIgYm9yZGVyPSIwIiBjZWxscGFkZGluZz0iMCIgY2VsbHNwYWNpbmc9IjAiIHdpZHRoPSIxMDAlIiBzdHlsZT0iYm9yZGVyLWNvbGxhcHNlOiBjb2xsYXBzZTt0YWJsZS1sYXlvdXQ6IGZpeGVkO2JvcmRlci1zcGFjaW5nOiAwO21zby10YWJsZS1sc3BhY2U6IDBwdDttc28tdGFibGUtcnNwYWNlOiAwcHQ7dmVydGljYWwtYWxpZ246IHRvcDtib3JkZXItdG9wOiAxcHggc29saWQgI0JCQkJCQjstbXMtdGV4dC1zaXplLWFkanVzdDogMTAwJTstd2Via2l0LXRleHQtc2l6ZS1hZGp1c3Q6IDEwMCUiPgogICAgPHRib2R5PgogICAgICA8dHIgc3R5bGU9InZlcnRpY2FsLWFsaWduOiB0b3AiPgogICAgICAgIDx0ZCBzdHlsZT0id29yZC1icmVhazogYnJlYWstd29yZDtib3JkZXItY29sbGFwc2U6IGNvbGxhcHNlICFpbXBvcnRhbnQ7dmVydGljYWwtYWxpZ246IHRvcDtmb250LXNpemU6IDBweDtsaW5lLWhlaWdodDogMHB4O21zby1saW5lLWhlaWdodC1ydWxlOiBleGFjdGx5Oy1tcy10ZXh0LXNpemUtYWRqdXN0OiAxMDAlOy13ZWJraXQtdGV4dC1zaXplLWFkanVzdDogMTAwJSI+CiAgICAgICAgICA8c3Bhbj4mIzE2MDs8L3NwYW4+CiAgICAgICAgPC90ZD4KICAgICAgPC90cj4KICAgIDwvdGJvZHk+CiAgPC90YWJsZT4KCiAgICAgIDwvdGQ+CiAgICA8L3RyPgogIDwvdGJvZHk+CjwvdGFibGU+Cgo8dGFibGUgc3R5bGU9ImZvbnQtZmFtaWx5OmFyaWFsLGhlbHZldGljYSxzYW5zLXNlcmlmOyIgcm9sZT0icHJlc2VudGF0aW9uIiBjZWxscGFkZGluZz0iMCIgY2VsbHNwYWNpbmc9IjAiIHdpZHRoPSIxMDAlIiBib3JkZXI9IjAiPgogIDx0Ym9keT4KICAgIDx0cj4KICAgICAgPHRkIHN0eWxlPSJvdmVyZmxvdy13cmFwOmJyZWFrLXdvcmQ7d29yZC1icmVhazpicmVhay13b3JkO3BhZGRpbmc6MTBweDtmb250LWZhbWlseTphcmlhbCxoZWx2ZXRpY2Esc2Fucy1zZXJpZjsiIGFsaWduPSJsZWZ0Ij4KICAgICAgICAKICA8ZGl2IHN0eWxlPSJmb250LXNpemU6IDE0cHg7IGxpbmUtaGVpZ2h0OiAxNDAlOyB0ZXh0LWFsaWduOiBjZW50ZXI7IHdvcmQtd3JhcDogYnJlYWstd29yZDsiPgogICAgPHAgc3R5bGU9ImxpbmUtaGVpZ2h0OiAxNDAlOyI+Q2xpY2sgb24gdGhlIGJ1dHRvbiBiZWxvdyB0byB2aXNpdCBvdXIgcGFnZS48L3A+CiAgPC9kaXY+CgogICAgICA8L3RkPgogICAgPC90cj4KICA8L3Rib2R5Pgo8L3RhYmxlPgoKPHRhYmxlIHN0eWxlPSJmb250LWZhbWlseTphcmlhbCxoZWx2ZXRpY2Esc2Fucy1zZXJpZjsiIHJvbGU9InByZXNlbnRhdGlvbiIgY2VsbHBhZGRpbmc9IjAiIGNlbGxzcGFjaW5nPSIwIiB3aWR0aD0iMTAwJSIgYm9yZGVyPSIwIj4KICA8dGJvZHk+CiAgICA8dHI+CiAgICAgIDx0ZCBzdHlsZT0ib3ZlcmZsb3ctd3JhcDpicmVhay13b3JkO3dvcmQtYnJlYWs6YnJlYWstd29yZDtwYWRkaW5nOjEwcHg7Zm9udC1mYW1pbHk6YXJpYWwsaGVsdmV0aWNhLHNhbnMtc2VyaWY7IiBhbGlnbj0ibGVmdCI+CiAgICAgICAgCiAgPCEtLVtpZiBtc29dPjxzdHlsZT4udi1idXR0b24ge2JhY2tncm91bmQ6IHRyYW5zcGFyZW50ICFpbXBvcnRhbnQ7fTwvc3R5bGU+PCFbZW5kaWZdLS0+CjxkaXYgYWxpZ249ImNlbnRlciI+CiAgPCEtLVtpZiBtc29dPjx2OnJvdW5kcmVjdCB4bWxuczp2PSJ1cm46c2NoZW1hcy1taWNyb3NvZnQtY29tOnZtbCIgeG1sbnM6dz0idXJuOnNjaGVtYXMtbWljcm9zb2Z0LWNvbTpvZmZpY2U6d29yZCIgaHJlZj0iaHR0cHM6Ly93d3cubWlycm9yY2RwLmNvbS8iIHN0eWxlPSJoZWlnaHQ6MzdweDsgdi10ZXh0LWFuY2hvcjptaWRkbGU7IHdpZHRoOjc3cHg7IiBhcmNzaXplPSIxMSUiICBzdHJva2U9ImYiIGZpbGxjb2xvcj0iIzAxNmZlZSI+PHc6YW5jaG9ybG9jay8+PGNlbnRlciBzdHlsZT0iY29sb3I6I0ZGRkZGRjsiPjwhW2VuZGlmXS0tPgogICAgPGEgaHJlZj0iaHR0cHM6Ly93d3cubWlycm9yY2RwLmNvbS8iIHRhcmdldD0iX2JsYW5rIiBjbGFzcz0idi1idXR0b24iIHN0eWxlPSJib3gtc2l6aW5nOiBib3JkZXItYm94O2Rpc3BsYXk6IGlubGluZS1ibG9jazt0ZXh0LWRlY29yYXRpb246IG5vbmU7LXdlYmtpdC10ZXh0LXNpemUtYWRqdXN0OiBub25lO3RleHQtYWxpZ246IGNlbnRlcjtjb2xvcjogI0ZGRkZGRjsgYmFja2dyb3VuZC1jb2xvcjogIzAxNmZlZTsgYm9yZGVyLXJhZGl1czogNHB4Oy13ZWJraXQtYm9yZGVyLXJhZGl1czogNHB4OyAtbW96LWJvcmRlci1yYWRpdXM6IDRweDsgd2lkdGg6YXV0bzsgbWF4LXdpZHRoOjEwMCU7IG92ZXJmbG93LXdyYXA6IGJyZWFrLXdvcmQ7IHdvcmQtYnJlYWs6IGJyZWFrLXdvcmQ7IHdvcmQtd3JhcDpicmVhay13b3JkOyBtc28tYm9yZGVyLWFsdDogbm9uZTtmb250LXNpemU6IDE0cHg7Ij4KICAgICAgPHNwYW4gc3R5bGU9ImRpc3BsYXk6YmxvY2s7cGFkZGluZzoxMHB4IDIwcHg7bGluZS1oZWlnaHQ6MTIwJTsiPjxzcGFuIHN0eWxlPSJsaW5lLWhlaWdodDogMTYuOHB4OyI+TWlycm9yPC9zcGFuPjwvc3Bhbj4KICAgIDwvYT4KICAgIDwhLS1baWYgbXNvXT48L2NlbnRlcj48L3Y6cm91bmRyZWN0PjwhW2VuZGlmXS0tPgo8L2Rpdj4KCiAgICAgIDwvdGQ+CiAgICA8L3RyPgogIDwvdGJvZHk+CjwvdGFibGU+CgogIDwhLS1baWYgKCFtc28pJighSUUpXT48IS0tPjwvZGl2PjwhLS08IVtlbmRpZl0tLT4KICA8L2Rpdj4KPC9kaXY+CjwhLS1baWYgKG1zbyl8KElFKV0+PC90ZD48IVtlbmRpZl0tLT4KICAgICAgPCEtLVtpZiAobXNvKXwoSUUpXT48L3RyPjwvdGFibGU+PC90ZD48L3RyPjwvdGFibGU+PCFbZW5kaWZdLS0+CiAgICA8L2Rpdj4KICA8L2Rpdj4KICA8L2Rpdj4KICAKCgogICAgPCEtLVtpZiAobXNvKXwoSUUpXT48L3RkPjwvdHI+PC90YWJsZT48IVtlbmRpZl0tLT4KICAgIDwvdGQ+CiAgPC90cj4KICA8L3Rib2R5PgogIDwvdGFibGU+CiAgPCEtLVtpZiBtc29dPjwvZGl2PjwhW2VuZGlmXS0tPgogIDwhLS1baWYgSUVdPjwvZGl2PjwhW2VuZGlmXS0tPgo8L2JvZHk+Cgo8L2h0bWw+Cg=="
)

var dummyEmailJson = fmt.Sprintf(`{"counters":{"u_column":1,"u_row":1,"u_content_text":3,"u_content_divider":1,"u_content_heading":1,"u_content_button":1},"body":{"id":"TZz18h5bCd","rows":[{"id":"lh5PZRVf3u","cells":[1],"columns":[{"id":"AA48ApALEi","contents":[{"id":"SCG_p4fO7n","type":"heading","values":{"containerPadding":"10px","anchor":"","headingType":"h1","fontSize":"22px","textAlign":"center","lineHeight":"140%","linkStyle":{"inherit":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_heading_1","htmlClassNames":"u_content_heading"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<span>Welcome to Mirror!</span>","_languages":{}}},{"id":"uiK4pRpetX","type":"divider","values":{"width":"100%","border":{"borderTopWidth":"1px","borderTopStyle":"solid","borderTopColor":"#BBBBBB"},"textAlign":"center","containerPadding":"10px","anchor":"","displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_divider_1","htmlClassNames":"u_content_divider"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true}},{"id":"81CDqbcmae","type":"text","values":{"containerPadding":"10px","fontSize":"14px","textAlign":"center","lineHeight":"140%","linkStyle":{"inherit":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"displayCondition":null,"_styleGuide":null,"_meta":{"htmlID":"u_content_text_3","htmlClassNames":"u_content_text"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<p style=\"line-height: 140%;\">Click on the button below to visit our page.</p>","_languages":{}}},{"id":"9qZJEn2ypl","type":"button","values":{"href":{"name":"web","attrs":{"href":"{{href}}","target":"{{target}}"},"values":{"href":"%s","target":"_blank"}},"buttonColors":{"color":"#FFFFFF","backgroundColor":"#016fee","hoverColor":"#FFFFFF","hoverBackgroundColor":"#3AAEE0"},"size":{"autoWidth":true,"width":"100%"},"fontSize":"14px","lineHeight":"120%","textAlign":"center","padding":"10px 20px","border":{},"borderRadius":"4px","displayCondition":null,"_styleGuide":null,"containerPadding":"10px","anchor":"","_meta":{"htmlID":"u_content_button_1","htmlClassNames":"u_content_button"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true,"text":"<span style=\"line-height: 16.8px;\">Mirror</span>","_languages":{},"calculatedWidth":77,"calculatedHeight":37}}],"values":{"backgroundColor":"","padding":"0px","border":{},"borderRadius":"0px","_meta":{"htmlID":"u_column_1","htmlClassNames":"u_column"}}}],"values":{"displayCondition":null,"columns":false,"_styleGuide":null,"backgroundColor":"","columnsBackgroundColor":"","backgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"custom","position":"center","customPosition":["50%","50%"]},"padding":"0px","anchor":"","hideDesktop":false,"_meta":{"htmlID":"u_row_1","htmlClassNames":"u_row"},"selectable":true,"draggable":true,"duplicatable":true,"deletable":true,"hideable":true}}],"headers":[],"footers":[],"values":{"_styleGuide":null,"popupPosition":"center","popupWidth":"600px","popupHeight":"auto","borderRadius":"10px","contentAlign":"center","contentVerticalAlign":"center","contentWidth":"500px","fontFamily":{"label":"Arial","value":"arial,helvetica,sans-serif"},"textColor":"#000000","popupBackgroundColor":"#FFFFFF","popupBackgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"cover","position":"center"},"popupOverlay_backgroundColor":"rgba(0, 0, 0, 0.1)","popupCloseButton_position":"top-right","popupCloseButton_backgroundColor":"#DDDDDD","popupCloseButton_iconColor":"#000000","popupCloseButton_borderRadius":"0px","popupCloseButton_margin":"0px","popupCloseButton_action":{"name":"close_popup","attrs":{"onClick":"document.querySelector('.u-popup-container').style.display = 'none';"}},"language":{},"backgroundColor":"#F7F8F9","preheaderText":"","linkStyle":{"body":true,"linkColor":"#0000ee","linkHoverColor":"#0000ee","linkUnderline":true,"linkHoverUnderline":true},"backgroundImage":{"url":"","fullWidth":true,"repeat":"no-repeat","size":"custom","position":"center"},"_meta":{"htmlID":"u_body","htmlClassNames":"u_body"}}},"schemaVersion":18}`, targetLink)

type AccountHandler interface {
	CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error
}

type accountHandler struct {
	cfg             *config.Config
	tenantHandler   TenantHandler
	userHandler     UserHandler
	tagHandler      TagHandler
	segmentHandler  SegmentHandler
	emailHandler    EmailHandler
	campaignRepo    repo.CampaignRepo
	queryRepo       repo.QueryRepo
	taskRepo        repo.TaskRepo
	campaignLogRepo repo.CampaignLogRepo
}

func NewAccountHandler(cfg *config.Config, userHandler UserHandler, tenantHandler TenantHandler,
	tagHandler TagHandler, segmentHandler SegmentHandler, emailHandler EmailHandler, campaignRepo repo.CampaignRepo,
	queryRepo repo.QueryRepo, taskRepo repo.TaskRepo, campaignLogRepo repo.CampaignLogRepo) AccountHandler {
	return &accountHandler{
		cfg:             cfg,
		tenantHandler:   tenantHandler,
		userHandler:     userHandler,
		tagHandler:      tagHandler,
		segmentHandler:  segmentHandler,
		emailHandler:    emailHandler,
		campaignRepo:    campaignRepo,
		queryRepo:       queryRepo,
		taskRepo:        taskRepo,
		campaignLogRepo: campaignLogRepo,
	}
}

type CreateTrialAccountRequest struct {
	Token *string `schema:"token,omitempty"`
}

func (r *CreateTrialAccountRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type CreateTrialAccountResponse struct {
	Session *entity.Session `json:"session,omitempty"`
}

var CreateTrialAccountValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *accountHandler) CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error {
	if err := CreateTrialAccountValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.GetToken() != h.cfg.TrialAccountToken {
		return errutil.ValidationError(errors.New("invalid trial account token"))
	}

	// ========== Create Tenant ==========

	var (
		suffix     = strings.ToLower(goutil.GenerateRandString(15))
		tenantName = fmt.Sprintf("demo-mirror-%s", suffix)
	)

	var (
		createTenantReq = &CreateTenantRequest{
			Name: goutil.String(tenantName),
		}
		createTenantRes = new(CreateTenantResponse)
	)
	if err := h.tenantHandler.CreateTenant(ctx, createTenantReq, createTenantRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create tenant error: %s", err)
		return err
	}

	// ========== Init Tenant with Admin User ==========

	password, err := goutil.GenerateSecureRandString(15)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("generate random password failed: %v", err)
		return err
	}

	var (
		username = "admin"
		email    = fmt.Sprintf("%s@mirror.com", username)

		initTenantReq = &InitTenantRequest{
			Token: createTenantRes.Token,
			User: &CreateUserRequest{
				Email:    goutil.String(email),
				Password: goutil.String(password),
			},
		}
		initTenantRes = new(InitTenantResponse)
	)
	if err := h.tenantHandler.InitTenant(ctx, initTenantReq, initTenantRes); err != nil {
		log.Ctx(ctx).Error().Msgf("init tenant error: %s", err)
		return err
	}

	// ========== Create Two Tags ==========

	var (
		tenant    = initTenantRes.Tenant
		adminUser = initTenantRes.Users[0]
	)

	contextInfo := ContextInfo{
		Tenant: tenant,
		User:   adminUser,
	}

	var (
		createTagReqs = []*CreateTagRequest{
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Age"),
				TagDesc:     goutil.String("Age of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeInt)),
			},
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Country"),
				TagDesc:     goutil.String("Country of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeStr)),
			},
		}
		createTagRess = []*CreateTagResponse{
			new(CreateTagResponse),
			new(CreateTagResponse),
		}
	)
	for i, tagReq := range createTagReqs {
		if err := h.tagHandler.CreateTag(ctx, tagReq, createTagRess[i]); err != nil {
			log.Ctx(ctx).Error().Msgf("create tag error: %s", err)
			return err
		}
	}

	// ========== Create Segment ==========

	var (
		ageTag     = createTagRess[0].Tag
		countryTag = createTagRess[1].Tag
	)

	var (
		createSegmentReq = &CreateSegmentRequest{
			ContextInfo: contextInfo,
			Name:        goutil.String("Millennials or Malaysians"),
			SegmentDesc: goutil.String("Users aged between 18 and 40, or users from Malaysia"),
			Criteria: &entity.Query{
				Queries: []*entity.Query{
					{
						Lookups: []*entity.Lookup{
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpGt,
								Val:   18,
							},
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpLt,
								Val:   40,
							},
						},
						Op: entity.QueryOpAnd,
					},
					{
						Lookups: []*entity.Lookup{
							{
								TagID: countryTag.ID,
								Op:    entity.LookupOpEq,
								Val:   "Malaysia",
							},
						},
						Op: entity.QueryOpAnd,
					},
				},
				Op: entity.QueryOpOr,
			},
		}
		createSegmentRes = new(CreateSegmentResponse)
	)
	if err := h.segmentHandler.CreateSegment(ctx, createSegmentReq, createSegmentRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create segment error: %s", err)
		return err
	}

	// ========== Create Dummy Tasks ==========

	var (
		now         = uint64(time.Now().Unix())
		userSize    = uint64(20)
		resourceIDs = []uint64{ageTag.GetID(), countryTag.GetID()}
		task        = &entity.Task{
			TenantID:     tenant.ID,
			ResourceID:   nil,
			Status:       entity.TaskStatusSuccess,
			TaskType:     entity.TaskTypeFileUpload,
			ResourceType: entity.ResourceTypeTag,
			ExtInfo: &entity.TaskExtInfo{
				FileID:      goutil.String(""),
				OriFileName: goutil.String("file.csv"),
				Size:        goutil.Uint64(userSize),
				Progress:    goutil.Uint64(100),
			},
			CreatorID:  adminUser.ID,
			CreateTime: goutil.Uint64(now),
			UpdateTime: goutil.Uint64(now),
		}
	)
	for _, resourceID := range resourceIDs {
		task.ResourceID = goutil.Uint64(resourceID)

		_, err := h.taskRepo.Create(ctx, task)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create task error: %s", err)
			return err
		}
	}

	// ========== Insert UD Tag Vals ==========

	dummyUserEmails := make([]string, 0)
	for i := 0; i < int(userSize); i++ {
		dummyUserEmails = append(dummyUserEmails, fmt.Sprintf("%s@mirror.com", goutil.GenerateRandString(10)))
	}

	udTagVals := make([]*entity.UdTagVal, 0)
	for _, dummyUserEmail := range dummyUserEmails {
		udTagVals = append(udTagVals, &entity.UdTagVal{
			Ud: &entity.Ud{
				ID:     goutil.String(dummyUserEmail),
				IDType: entity.IDTypeEmail,
			},
			TagVals: []*entity.TagVal{
				{
					TagID:  ageTag.ID,
					TagVal: goutil.Uint64(uint64(rand.Intn(48-18+1) + 18)), // between 18 and 48
				},
				{
					TagID:  countryTag.ID,
					TagVal: goutil.String("Malaysia"),
				},
			},
		})
	}

	if err := h.queryRepo.BatchUpsert(ctx, tenantName, udTagVals); err != nil {
		log.Ctx(ctx).Error().Msgf("batch upsert error: %v", err)
		return err
	}

	// ========== Create Email ==========

	var (
		createEmailReq = &CreateEmailRequest{
			ContextInfo: contextInfo,
			Name:        goutil.String("Sample Email"),
			EmailDesc:   goutil.String("This is a sample email"),
			Html:        goutil.String(dummyEmailHtmlEncoded),
			Json:        goutil.String(dummyEmailJson),
		}
		createEmailRes = new(CreateEmailResponse)
	)
	if err := h.emailHandler.CreateEmail(ctx, createEmailReq, createEmailRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create email error: %s", err)
		return err
	}

	// ========== Create Campaign ==========

	campaign := &entity.Campaign{
		Name:         goutil.String("Sample Campaign"),
		CampaignDesc: goutil.String("This is a sample campaign."),
		SegmentID:    createSegmentRes.Segment.ID,
		SegmentSize:  goutil.Uint64(userSize),
		Progress:     goutil.Uint64(100),
		Status:       entity.CampaignStatusRunning,
		CampaignEmails: []*entity.CampaignEmail{
			{
				EmailID: createEmailRes.Email.ID,
				Subject: goutil.String("A/B Test Subject #1"),
				Ratio:   goutil.Uint64(50),
				CampaignResult: &entity.CampaignResult{
					TotalUniqueOpenCount: goutil.Uint64(5),
					TotalClickCount:      goutil.Uint64(3),
					AvgOpenTime:          goutil.Uint64(now + 3600),
					ClickCountsByLink: map[string]uint64{
						targetLink: 3,
					},
				},
			},
			{
				EmailID: createEmailRes.Email.ID,
				Subject: goutil.String("A/B Test Subject #2"),
				Ratio:   goutil.Uint64(50),
				CampaignResult: &entity.CampaignResult{
					TotalUniqueOpenCount: goutil.Uint64(10),
					TotalClickCount:      goutil.Uint64(5),
					AvgOpenTime:          goutil.Uint64(now + 7200),
					ClickCountsByLink: map[string]uint64{
						targetLink: 5,
					},
				},
			},
		},
		CreatorID:  adminUser.ID,
		TenantID:   tenant.ID,
		Schedule:   goutil.Uint64(now),
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}

	_, err = h.campaignRepo.Create(ctx, campaign)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign error: %s", err)
		return err
	}

	// ========== Create Campaign Logs ==========

	campaignLogs := make([]*entity.CampaignLog, 0)
	for i := 0; i < 2; i++ {
		for j := 0; j < i+3; j++ {
			campaignLogs = append(campaignLogs, &entity.CampaignLog{
				CampaignEmailID: goutil.Uint64(uint64(i + 1)),
				Event:           entity.EventUniqueOpened,
				Link:            goutil.String(""),
				EventTime:       goutil.Uint64(now + 3600),
				Email:           goutil.String(dummyUserEmails[j]),
				CreateTime:      goutil.Uint64(now + 3600),
			})
		}

		for j := 0; j < i+2; j++ {
			campaignLogs = append(campaignLogs, &entity.CampaignLog{
				CampaignEmailID: goutil.Uint64(uint64(i + 1)),
				Event:           entity.EventClick,
				Link:            goutil.String(targetLink),
				EventTime:       goutil.Uint64(now + 3600),
				Email:           goutil.String(dummyUserEmails[j]),
				CreateTime:      goutil.Uint64(now + 3600),
			})
		}
	}

	if err := h.campaignLogRepo.CreateMany(ctx, campaignLogs); err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign logs error: %v", err)
		return err
	}

	// ========== Log In ==========

	var (
		logInReq = &LogInRequest{
			TenantName: goutil.String(tenantName),
			Username:   goutil.String(username),
			Password:   goutil.String(password),
		}
		logInRes = new(LogInResponse)
	)
	if err := h.userHandler.LogIn(ctx, logInReq, logInRes); err != nil {
		log.Ctx(ctx).Error().Msgf("login user error: %s", err)
		return err
	}

	res.Session = logInRes.Session

	return nil
}
