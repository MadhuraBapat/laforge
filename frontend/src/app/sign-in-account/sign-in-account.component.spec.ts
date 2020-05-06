import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SignInAccountComponent } from './sign-in-account.component';

describe('SignInAccountComponent', () => {
  let component: SignInAccountComponent;
  let fixture: ComponentFixture<SignInAccountComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SignInAccountComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SignInAccountComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
